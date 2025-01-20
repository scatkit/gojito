package rpc
import(
  "context"
  "github.com/scatkit/gojito/rpc/jitorpc"
)

type JITORPC interface{
  MakeCall(ctx context.Context, path string, RPCPayload *jitorpc.RPCPayload) (*jitorpc.RPCResponse, error)
  MakeCallWithHeader(ctx context.Context, path string, RPCPayload *jitorpc.RPCPayload) (*jitorpc.RPCResponseWithHeader, error)
}

type JitoClient struct{
  jitoURL     string 
  jitoRPC     JITORPC    
  uuid        string
}

func NewJito(endpoint, uuid string) *JitoClient{
  jitoRPC := jitorpc.NewClient(endpoint)
  return &JitoClient{
    jitoURL:  endpoint,
    jitoRPC:  jitoRPC,
    uuid:     uuid,
  }
}
