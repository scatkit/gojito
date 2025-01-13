package searcher_client
import (
  "context"
  "crypto/tls"
  "net"
  "net/http"
  "encoding/base64"
  "time"
  "strings"
  "fmt"
  "bufio"
  
  "github.com/scatkit/pumpdexer/rpc"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/gojito/pb"
  "github.com/scatkit/gojito/pkg"
  
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials"
  "google.golang.org/grpc/keepalive"
)

type Client struct{
  GrpcConn    *grpc.ClientConn
  RpcConn     *rpc.Client // executes standard Solana's RPC requests
  JitoRpcConn *rpc.Client//  executes specific JITO RPC requests
  SearcherService            jito_pb.SearcherServiceClient
  BundleStreamSubscription   jito_pb.SearcherService_SubscribeBundleResultsClient // Used for receiving *jito_pb.BundleResult (bundle broadcast status info).
  Auth *pkg.AuthenticationService 
  ErrChan chan error // ErrChan is used for dispatching errors from functions executed within goroutines.
}

// Creates a New Searcher client instance
func New(
  ctx context.Context,
  grpcDialURL string,
  jitoRpcClient, rpcClient *rpc.Client,
  privateKey solana.PrivateKey, // for authentication
  tlsConfig *tls.Config,
  opts ...grpc.DialOption,
) (*Client, error){
  
  // configure gRPC transport credentials
  if tlsConfig != nil{
    opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))) // custom tlsConfig
  } else{
    opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
  }
  
  // Ensure error monitoring for observable gRPC connection
  chErr := make(chan error)
  conn, err := pkg.CreateAndObserveGRPCConn(ctx, chErr, grpcDialURL, opts...)
  if err != nil{
    return nil, err
  }
  
  // New searcher client allows to ineract with searcher-realated APIs (e.g sending bundles)
  searcherService := jito_pb.NewSearcherServiceClient(conn) 
  authService := pkg.NewAuthenticationService(conn, privateKey)
  
  // Authenticates the client with a specified role
  if err = authService.AuthenticateAndRefresh(jito_pb.Role_SEARCHER); err != nil{
    return nil, err
  }
  
  subBundleRes, err := searcherService.SubscribeBundleResults(authService.GrpcCtx, &jito_pb.SubscribeBundleResultsRequest{})
	if err != nil {
		return nil, err
	}
  
  return &Client{
    GrpcConn: conn,
    RpcConn: rpcClient,
    JitoRpcConn: jitoRpcClient,
    SearcherService: searcherService,
    BundleStreamSubscription: subBundleRes,
    Auth: authService,
    ErrChan: chErr,
  }, nil
}

func (cl *Client) Close() error{
  close(cl.ErrChan)
  defer cl.Auth.GrpcCtx.Done()

  if err := cl.RpcConn.Close(); err != nil{ // Closing pumpdexer rpc client for Solana 
    return err
  }
  if err := cl.JitoRpcConn.Close(); err != nil{ // Closing punodexer rpc client for Jito
    return err   
  }

  return cl.GrpcConn.Close()
}


// NewNoAuth initializes and retunres a new instance of a Searcher client which doesn't require private key signing
func NewNoAuth(
  ctx context.Context,
  grpcDialURL string,
  jitoRpcClient, rpcClient *rpc.Client,
  tlsConfig *tls.Config,
  proxyURL string,
  opts ...grpc.DialOption,
) (*Client, error){

  if tlsConfig != nil {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

  if proxyURL != ""{
    dialer, err := createContextDialer(proxyURL)
    if err != nil {
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}
    opts = append(opts,
			grpc.WithContextDialer(dialer),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                15 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			}),
		)
  }
  
  chErr := make(chan error)
	conn, err := pkg.CreateAndObserveGRPCConn(ctx, chErr, grpcDialURL, opts...)
	if err != nil {
		return nil, err
	}
  
  searcherService := jito_pb.NewSearcherServiceClient(conn)
	subBundleRes, err := searcherService.SubscribeBundleResults(ctx, &jito_pb.SubscribeBundleResultsRequest{})
  
  return &Client{
		GrpcConn:                 conn,
		RpcConn:                  rpcClient,
		JitoRpcConn:              jitoRpcClient,
		SearcherService:          searcherService,
		BundleStreamSubscription: subBundleRes,
		ErrChan:                  chErr,
		Auth:                     &pkg.AuthenticationService{GrpcCtx: ctx},
	}, nil
}

func createContextDialer(proxyStr string) (func(context.Context, string) (net.Conn, error), error) {
	pd, err := newProxyDialer(proxyStr)
	if err != nil {
		return nil, err
	}

	return pd.dialProxy, nil
}

type proxyDialer struct {
  proxyHost string
  auth      string
  timeout   time.Duration
}

func (d *proxyDialer) dialProxy(ctx context.Context, addr string) (net.Conn, error) {
	var conn net.Conn
	var err error

	dialer := &net.Dialer{
		Timeout:   d.timeout,
		KeepAlive: 30 * time.Second,
	}

	conn, err = dialer.DialContext(ctx, "tcp", d.proxyHost)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy %s: %w", d.proxyHost, err)
	}

	connectReq := fmt.Sprintf(
		"CONNECT %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Proxy-Authorization: Basic %s\r\n"+
			"User-Agent: Go-http-client/1.1\r\n"+
			"\r\n",
		addr, addr, d.auth,
	)

	if _, err = conn.Write([]byte(connectReq)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write CONNECT request: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("proxy connection failed: %s", resp.Status)
	}

	return conn, nil
}

func newProxyDialer(proxyStr string) (*proxyDialer, error) {
	host, port, username, password, err := parseProxyString(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy string: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	return &proxyDialer{
		proxyHost: net.JoinHostPort(host, port),
		auth:      auth,
		timeout:   30 * time.Second,
	}, nil
}

func parseProxyString(proxyStr string) (host string, port string, username string, password string, err error) {
	parts := strings.Split(proxyStr, ":")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid proxy format, expected IP:PORT:USERNAME:PASSWORD")
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}

func isRPCNil(client *rpc.Client){
  if client == nil{
    client = rpc.New("https://api.mainnet-beta.solana.com")
  }
}
