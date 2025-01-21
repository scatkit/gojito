package jitorpc
import(
  "context"
  "fmt" 
  "encoding/json"
  "encoding/base64"
  "strings"
  
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/gojito/jitorpc/jsonrpc"
  "github.com/davecgh/go-spew/spew"

)

//type TransactionResponse struct{
//	Jsonrpc  string `json:"jsonrpc"`
//	Result   string `json:"result"`
//	ID       int    `json:"id"`
//	BundleID string
//}

func (cl *JitoClient) SendTransaction(ctx context.Context, signedTx *solana.Transaction, bundleOnly bool,
) (txSig solana.Signature, err error){
  // TO-DO: this is Legacy
  encodedTx, err := signedTx.MarshalBinary()
  if err != nil{
    return solana.Signature{}, err
  }
  
  base64Transaction := base64.StdEncoding.EncodeToString(encodedTx)
  params := []interface{}{base64Transaction, map[string]string{"encoding": "base64"}}
  
  payload := &jsonrpc.RPCPayload{
    JSONRPC: "2.0",
    Method: "sendTransaction", 
    Params: params, 
  }
  
  var path = "/api/v1/transactions"
  queryParams := []string{}

	if bundleOnly {
		queryParams = append(queryParams, "bundleOnly=true")
	}
	if cl.uuid != "" {
		queryParams = append(queryParams, fmt.Sprintf("uuid=%s", cl.uuid))
	}

	if len(queryParams) > 0 {
		path = fmt.Sprintf("%s?%s", path, strings.Join(queryParams, "&"))
	}
  
  resp, err := cl.jitoRPC.MakeCallWithHeader(ctx, path, payload)
  spew.Dump(resp)
  err = json.Unmarshal(resp.Result, &txSig)
  BundleID := resp.BundleID 
  fmt.Println(BundleID)
  
  return txSig, err
}
