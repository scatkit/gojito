package rpc
import(
  "context"
  "fmt" 
  "encoding/json"
  "strings"
  
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/gojito/rpc/jitorpc"
)

type TransactionResponse struct{
	Jsonrpc  string `json:"jsonrpc"`
	Result   string `json:"result"`
	ID       int    `json:"id"`
	BundleID string
}

func (cl *JitoClient) SendTransaction(ctx context.Context, signedTx *solana.Transaction, bundleOnly bool,
) (txResponse *TransactionResponse, err error){
  params := []interface{}{signedTx, map[string]string{"encoding": "base64"}}
  
  payload := &jitorpc.RPCPayload{
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
  err = json.Unmarshal(resp.Result, &txResponse)
  txResponse.BundleID = resp.BundleID 
  
  return txResponse, err
}
