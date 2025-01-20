package searcher_client
import (
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/pumpdexer/programs/system"
  "github.com/scatkit/gojito/pb"
  "google.golang.org/grpc"
  "math/rand"
)

func (cl *Client) GenerateTipRandomAccountInstruction(tipAmount uint64, from solana.PublicKey) (solana.Instruction, error) {
	tipAccount, err := cl.GetRandomTipAccount() // baes58 string
	if err != nil {
		return nil, err
	}

	return system.NewTransferInstruction(tipAmount, from, solana.MustPubkeyFromBase58(tipAccount)).Build(), nil
}

func (cl *Client) GetRandomTipAccount(opts ...grpc.CallOption) (string, error){
  resp, err := cl.GetTipAccounts(opts...)
  if err != nil{
    return "", err
  } 
  
  return resp.Accounts[rand.Intn(len(resp.Accounts))], nil
}
 
func (cl *Client) GetTipAccounts(opts ...grpc.CallOption) (*jito_pb.GetTipAccountsResponse, error){
  return cl.SearcherService.GetTipAccounts(cl.Auth.GrpcCtx, &jito_pb.GetTipAccountsRequest{}, opts...)
}

