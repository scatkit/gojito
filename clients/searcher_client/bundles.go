package searcher_client
import (
  "errors"
  "context"
  "math/big"
  "fmt"
  "time"
  
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/pumpdexer/rpc"
  "github.com/scatkit/gojito/pb"
  "github.com/scatkit/gojito/pkg"
  
  "google.golang.org/grpc"
)

type Account struct {
	Executable bool     `json:"executable"`
	Owner      string   `json:"owner"`
	Lamports   int      `json:"lamports"`
	Data       []string `json:"data"`
	RentEpoch  *big.Int `json:"rentEpoch,omitempty"`
}

type TransactionResult struct {
	Err                   interface{} `json:"err,omitempty"`
	Logs                  []string    `json:"logs,omitempty"`
	PreExecutionAccounts  []Account   `json:"preExecutionAccounts,omitempty"`
	PostExecutionAccounts []Account   `json:"postExecutionAccounts,omitempty"`
	UnitsConsumed         *int        `json:"unitsConsumed,omitempty"`
	ReturnData            *ReturnData `json:"returnData,omitempty"`
}

type ReturnData struct {
	ProgramId string    `json:"programId"`
	Data      [2]string `json:"data"`
}

type SimulateBundleParams struct {
	EncodedTransactions []string `json:"encodedTransactions"`
}

type SimulateBundleConfig struct {
	PreExecutionAccountsConfigs  []ExecutionAccounts `json:"preExecutionAccountsConfigs"`
	PostExecutionAccountsConfigs []ExecutionAccounts `json:"postExecutionAccountsConfigs"`
}

type ExecutionAccounts struct {
	Encoding  string   `json:"encoding"`
	Addresses []string `json:"addresses"`
}

type SimulatedBundleResponse struct {
	Context interface{}                   `json:"context"`
	Value   SimulatedBundleResponseStruct `json:"value"`
}

type SimulatedBundleResponseStruct struct {
	Summary           interface{}         `json:"summary"`
	TransactionResult []TransactionResult `json:"transactionResults"`
}

// SimulateBundle is an RPC method that simulates a Jito bundle – exclusively available to Jito-Solana validator.
func (cl *Client) SimulateBundle(ctx context.Context, bundleParams SimulateBundleParams, simulationConfigs SimulateBundleConfig) (*SimulatedBundleResponse, error) {
	if len(bundleParams.EncodedTransactions) != len(simulationConfigs.PreExecutionAccountsConfigs) {
		return nil, errors.New("pre/post execution account config length must match bundle length")
	}
	var out SimulatedBundleResponse
  params := []interface{}{
    bundleParams, 
    simulationConfigs,
  }
	err := cl.JitoRpcConn.RPCCallForInfo(ctx, &out, "simulateBundle", params)
	return &out, err
}

// BroadcastBundleWithConfirmation sends a bundle of transactions on chain thru Jito BlockEngine and waits for its confirmation.
func (cl *Client) BroadcastBundleWithConfirmation(ctx context.Context, transactions []*solana.Transaction, opts ...grpc.CallOption, 
) (*jito_pb.SendBundleResponse, error){
  bundle, err := cl.BroadcastBundle(transactions, opts...)
  if err != nil{
    return nil, fmt.Errorf("Couldn't broadcast bundles: %w", err)
  }
  
  bundleSignatures := pkg.BatchExtractSigFromTx(transactions)
  
  for{
    select{
    case <- cl.Auth.GrpcCtx.Done():
      return nil, cl.Auth.GrpcCtx.Err()
    default:
      time.Sleep(5 * time.Second)
      bundleResult, err := cl.BundleStreamSubscription.Recv() // 
      if err != nil{
        return bundle, err
      }
      
      // (bundleResult, bundleID)
      if err := handleBundleResult(bundleResult, ""); err != nil{
        return bundle, err
      }
      
      //var start = time.Now()
      ctx, cancel := context.WithTimeout(ctx, time.Second*15)
      defer cancel()
      var statuses *rpc.GetSignatureStatusesResult
      
      isRPCNil(cl.RpcConn)
    
      for{
        // GetSignatureStatuses(context, searchTransactionHistory, transactionSignatures)
        statuses, err = cl.RpcConn.GetSignatureStatuses(ctx, false, bundleSignatures...)
        if err != nil{
          return bundle, err
        }
        ready := true
        for _, status := range statuses.Value{
          if status == nil{
            ready = false
            break
          }
        }
        if ready{
          break
        }
        select{
        case <- ctx.Done():
          return bundle, errors.New("operation timied out after 15 seconds")
        default:
          time.Sleep(1*time.Second)
        }
        //if time.Since(start) > time.Second*15{
        //  return bundle, errors.New("operation timed out after 15 secodns")
        //} else{
        //  time.Sleep(time.Second*1)
        //}
      }
      
      for _,status := range statuses.Value{
        if status.ConfirmationStatus != rpc.ConfirmationStatusProcessed && status.ConfirmationStatus != rpc.ConfirmationStatusConfirmed{
          return bundle, errors.New("searcher service did not provide bundle status in time")
        }
      }
      
      return bundle, nil
    }
  }
}

 
// Sends a bundle of transaction(s) on chain through Jito
func (cl *Client) BroadcastBundle(transactions []*solana.Transaction, opts ...grpc.CallOption) (*jito_pb.SendBundleResponse, error){
  bundle, err := cl.AssembleBundle(transactions) // array of protobuf packets
  if err != nil{
    return nil, err
  }
  
  return cl.SearcherService.SendBundle(cl.Auth.GrpcCtx, &jito_pb.SendBundleRequest{Bundle: bundle}, opts...)
}

// Converts an array of SOL transactions to a Jito bundle
func (cl *Client) AssembleBundle(transactions []*solana.Transaction) (*jito_pb.Bundle, error){
  packets := make([]*jito_pb.Packet, 0, len(transactions)) // <-- packets are encoded repr of srucutures data
  
  // converts an array of transactions to an array of protobuf packets
  for i, tx := range transactions{
    packet, err := pkg.ConvertTransactionToProtobufPacket(tx)
    if err != nil{
      return nil, fmt.Errorf("%d: error converting tx to jito_pb packet [%w]",i, err)
    }
    packets = append(packets, &packet)
  }
  
  return &jito_pb.Bundle{Packets: packets, Header: nil}, nil
}
 
// bundleID arg is solely for JSON RPC API.
func handleBundleResult[T *GetInflightBundlesStatusesResponse | *jito_pb.BundleResult](t T, bundleID string) error {
	switch bundle := any(t).(type) {
	case *jito_pb.BundleResult:
		switch bundle.Result.(type) {
		case *jito_pb.BundleResult_Accepted:
			break
		case *jito_pb.BundleResult_Rejected:
			rejected := bundle.Result.(*jito_pb.BundleResult_Rejected)
			switch rejected.Rejected.Reason.(type) {
			case *jito_pb.Rejected_SimulationFailure:
				rejection := rejected.Rejected.GetSimulationFailure()
				return NewSimulationFailureError(rejection.TxSignature, rejection.GetMsg())
			case *jito_pb.Rejected_StateAuctionBidRejected:
				rejection := rejected.Rejected.GetStateAuctionBidRejected()
				return NewStateAuctionBidRejectedError(rejection.AuctionId, rejection.SimulatedBidLamports)
			case *jito_pb.Rejected_WinningBatchBidRejected:
				rejection := rejected.Rejected.GetWinningBatchBidRejected()
				return NewWinningBatchBidRejectedError(rejection.AuctionId, rejection.SimulatedBidLamports)
			case *jito_pb.Rejected_InternalError:
				rejection := rejected.Rejected.GetInternalError()
				return NewInternalError(rejection.Msg)
			case *jito_pb.Rejected_DroppedBundle:
				rejection := rejected.Rejected.GetDroppedBundle()
				return NewDroppedBundle(rejection.Msg)
			default:
				return nil
			}
		}
	case *GetInflightBundlesStatusesResponse: // experimental, subject to changes
		for i, value := range bundle.Result.Value {
			if value.BundleId == bundleID {
				switch value.Status {
				case "Invalid":
					return fmt.Errorf("bundle %d is invalid", i)
				case "Pending":
					return fmt.Errorf("bundle %d is pending", i)
				case "Failed":
					return fmt.Errorf("bundle %d failed to land", i)
				case "Landed":
					return nil
				default:
					return fmt.Errorf("bundle %d unknown error", i)
				}
			}
		}
	}
	return nil
}

type GetInflightBundlesStatusesResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Context struct {
			Slot int `json:"slot"`
		} `json:"context"`
		Value []struct {
			BundleId   string      `json:"bundle_id"`
			Status     string      `json:"status"`
			LandedSlot interface{} `json:"landed_slot"`
		} `json:"value"`
	} `json:"result"`
	Id int `json:"id"`
}
