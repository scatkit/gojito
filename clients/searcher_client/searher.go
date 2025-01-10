package searcher_client
import (
  "context"
  "crypto/tls"
  
  "github.com/scatkit/pumpdexer/rpc"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/gojito/pb"
  "github.com/scatkit/gojito/pkg"
  
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials"
)

type Client struct{
  GrpcConn    *grpc.ClientConn
  RpcConn     *rpc.Client // executes standard Solana's RPC requests
  JitoRpcConn *rpc.Client//  executes specific JITO RPC requests
  SearcherService            jito_pb.SearcherServiceClient
  BundleStreamSubscription  jito_pb.SearcherService_SubscribeBundleResultsClient // Used for receiving *jito_pb.BundleResult (bundle broadcast status info).
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

