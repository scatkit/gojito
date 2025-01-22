package jitorpc
import(
  "context"
  "encoding/json"
  "fmt"
  "github.com/scatkit/gojito/jitorpc/jsonrpc"
)

type GetInflightBundleStatusesResponse struct {
		Context struct {
			Slot int `json:"slot"`
		} `json:"context"`
		Value []struct {
			BundleId   string      `json:"bundle_id"`
			Status     string      `json:"status"`
			LandedSlot uint64      `json:"landed_slot"`
		} `json:"value"`
}

func (cl *JitoClient) GetInflightBundleStatuses(ctx context.Context, bundleIDs []string, 
) (out *GetInflightBundleStatusesResponse, err error){
  
  path := "/api/v1/bundles"
  if cl.uuid != "" {
    path = fmt.Sprintf("%s?uuid=%s", path, cl.uuid)
  }
  
  payload := &jsonrpc.RPCPayload{
    JSONRPC: "2.0",
    Method: "getInflightBundleStatuses", 
    Params: [][]string{
      bundleIDs,
    },
  }
  
  resp, err := cl.jitoRPC.MakeCall(ctx, path, payload)
  if err != nil{
    return nil, err
  }
  
  err = json.Unmarshal(resp.Result, &out)
  return
}



