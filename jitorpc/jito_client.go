package jitorpc
import(
  //"io"
  "context"
  "github.com/scatkit/gojito/jitorpc/jsonrpc"
)

type JITORPC interface{
  MakeCall(ctx context.Context, path string, RPCPayload *jsonrpc.RPCPayload) (*jsonrpc.RPCResponse, error)
  MakeCallWithHeader(ctx context.Context, path string, RPCPayload *jsonrpc.RPCPayload) (*jsonrpc.RPCResponseWithHeader, error)
}

type JitoClient struct{
  jitoURL     string 
  jitoRPC     JITORPC    
  uuid        string
}

func NewJito(endpoint, uuid string) *JitoClient{
  jitoRPC := jsonrpc.NewClient(endpoint)
  return &JitoClient{
    jitoURL:  endpoint,
    jitoRPC:  jitoRPC,
    uuid:     uuid,
  }
}

//func (cl *JitoClient) Close() error {
//	if cl.jitoRPC == nil {
//		return nil
//	}
//  // See if rpcClient implements Close()
//	if c, ok := cl.jitoRPC.(io.Closer); ok {
//		return c.Close()
//	}
//	return nil
//}
