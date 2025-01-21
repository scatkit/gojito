package jitorpc
import(
  "fmt"
  "encoding/json"
  "context"
  "math/rand"
  
  "github.com/scatkit/pumpdexer/programs/system"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/gojito/jitorpc/jsonrpc"
)

 
func (cl *JitoClient) GetTipAccounts(ctx context.Context) ([]string, error){
  
  payload := &jsonrpc.RPCPayload{
    Method: "getTipAccounts", 
    JSONRPC: "2.0",
    Params: []interface{}{},
  }
  
  path := "/api/v1/bundles"
	if cl.uuid != "" {
		path = fmt.Sprintf("%s?uuid=%s", path, cl.uuid)
	}
  
  resp, err := cl.jitoRPC.MakeCall(ctx, path, payload)
  if err != nil{
    return nil, err
  }
  
  var tipAccounts []string
  if err := json.Unmarshal(resp.Result, &tipAccounts); err != nil{
    return nil, fmt.Errorf("failed to unmarshal tip accpounts")
  }
  
  if len(tipAccounts) <= 0{
    return nil, fmt.Errorf("Something went wrong: received [] tip accounts")
  }
  
  return tipAccounts, nil
}

type TipAccount struct{
  Address solana.PublicKey `json:"address"`
}

func (cl *JitoClient) GetRandomTipAccount(ctx context.Context) (*TipAccount, error){
  tipAccounts, err  := cl.GetTipAccounts(ctx)
  if err != nil{
    return nil, err
  }
  
  randIndex := rand.Intn(len(tipAccounts))
  return &TipAccount{Address: solana.MustPubkeyFromBase58(tipAccounts[randIndex])}, nil
}
 
func (cl *JitoClient) GenerateJitoTipInstruction(ctx context.Context, tipAmount uint64, fromWallet solana.PublicKey,
) (output solana.Instruction, err error){
  
  tipAccount, err := cl.GetRandomTipAccount(ctx)
  if err != nil{
    return nil, err
  }

  return system.NewTransferInstruction(tipAmount, fromWallet, tipAccount.Address).Build(), nil
}
