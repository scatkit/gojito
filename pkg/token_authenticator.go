package pkg
import(
  "github.com/scatkit/gojito/pb"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/mr-tron/base58"
  "context"
  "sync"
  "time"
  "fmt"
  "google.golang.org/grpc"
  "google.golang.org/grpc/metadata"
)

type AuthenticationService struct{
  AuthService   jito_pb.AuthServiceClient
  GrpcCtx       context.Context
  KeyPair       *Keypair
  BearerToken   string
  ExpiresAt     int64 // seconds
  ErrChan       chan error
  mu            sync.Mutex
} 

func NewAuthenticationService(grpcConn *grpc.ClientConn, privateKey solana.PrivateKey) *AuthenticationService{
  return &AuthenticationService{
    GrpcCtx:      context.Background(), 
    AuthService:  jito_pb.NewAuthServiceClient(grpcConn),
    KeyPair:      NewKeyPair(privateKey),
    ErrChan:      make(chan error),
    mu:           sync.Mutex{},
  }
}

func (as *AuthenticationService) AuthenticateAndRefresh(role jito_pb.Role) error {
  // The challenge is a server-generated string that the client must sign to prove ownership of the corresponding private key. 
	respChallenge, err := as.AuthService.GenerateAuthChallenge(as.GrpcCtx,
		&jito_pb.GenerateAuthChallengeRequest{
			Role:   role,
			Pubkey: as.KeyPair.PubKey.Bytes(),
		},
	)
	if err != nil {
		return err
	}

  // Combine the public key and server-provided challenge to get challenge string.
	challenge := fmt.Sprintf("%s-%s", as.KeyPair.PubKey.String(), respChallenge.GetChallenge())

  // Sign the challenge with the private key, producing a cryptographic signature (sig).
	sig, err := as.generateChallengeSignature([]byte(challenge))
	if err != nil {
		return err
	}

  // Sends the signed challenge to the server to request authentication tokens (e.g. an access token and a refresh token).
	respToken, err := as.AuthService.GenerateAuthTokens(as.GrpcCtx, &jito_pb.GenerateAuthTokensRequest{
		Challenge:       challenge,
		SignedChallenge: sig,
		ClientPubkey:    as.KeyPair.PubKey.Bytes(),
	})
	if err != nil {
		return err
	}

  // UpdateAuthorizationMetadata updates headers of the gRPC connection.
	as.updateAuthorizationMetadata(respToken.AccessToken)

	go func() {
		for {
			select {
			case <-as.GrpcCtx.Done():
				as.ErrChan <- as.GrpcCtx.Err()
			default:
				var resp *jito_pb.RefreshAccessTokenResponse
				resp, err = as.AuthService.RefreshAccessToken(as.GrpcCtx, &jito_pb.RefreshAccessTokenRequest{
					RefreshToken: respToken.RefreshToken.Value,
				})
				if err != nil {
					as.ErrChan <- fmt.Errorf("failed to refresh access token: %w", err)
					continue
				}

				as.updateAuthorizationMetadata(resp.AccessToken)
				time.Sleep(time.Until(resp.AccessToken.ExpiresAtUtc.AsTime()) - 15*time.Second)
			}
		}
	}()

	return nil
}

func (as *AuthenticationService) generateChallengeSignature(challenge []byte) ([]byte, error) {
	sig, err := as.KeyPair.PrivKey.Sign(challenge)
	if err != nil {
		return nil, err
	}

	return base58.Decode(sig.String())
}

// updateAuthorizationMetadata updates headers of the gRPC connection.
func (as *AuthenticationService) updateAuthorizationMetadata(token *jito_pb.Token) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.GrpcCtx = metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token.Value))
	as.BearerToken = token.Value
	as.ExpiresAt = token.ExpiresAtUtc.Seconds
}

