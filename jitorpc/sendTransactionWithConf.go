package jitorpc
import(
  "fmt"
  "context"
  "time"
  "errors"
  "github.com/scatkit/pumpdexer/solana"
  //"github.com/scatkit/pumpdexer/rpc"
  //"github.com/davecgh/go-spew/spew"
  //"github.com/scatkit/gojito/pkg"
)

type ConfirmedTxAsBundleResponse struct{
  txSig solana.Signature
}

func (cl *JitoClient) SendTransactionWithConf(ctx context.Context, signedTx *solana.Transaction,
) (out *JitoTxResponse, err error){
  // Should i send different ctx???
  bundleTxResp, err := cl.SendTransaction(ctx, signedTx, true)
  if err != nil{
    return nil, err
  }
  
  //bundleSignatures := pkg.BatchExtractSigFromTx(signedTx)
  
  //start := time.Now()
  poolInterval := 5 * time.Second
  //timeout := 15 * time.Second
  maxRetries := 60
  for attempt := 1; attempt <= maxRetries; attempt++{
    select{
    case <-ctx.Done():
      return nil, ctx.Err()
    default:
      // have to stricter the context
      bundleStatuses, err := cl.GetInflightBundleStatuses(context.Background(), []string{bundleTxResp.bundleID})
      if err != nil{
        return bundleTxResp, err
      }
      
      for i, value := range bundleStatuses.Value{
        if value.BundleId == bundleTxResp.bundleID{
          switch value.Status{
          case "Invalid":
            fmt.Printf("bundle %d is invalid: %s\n",i,bundleTxResp.bundleID)
          case "Pending":
            fmt.Printf("bundle %d is pending: %s\n", i, bundleTxResp.bundleID)
          case "Failed":
            fmt.Printf("bundle %d failed to land: %s\n", i, bundleTxResp.bundleID)
          case "Landed":
            fmt.Printf("bundle %d has landed: %s\n", i, bundleTxResp.bundleID)
            solScan := fmt.Sprintf("https://solscan.io/tx/%s", bundleTxResp.txSig)
            fmt.Println(solScan)
            return bundleTxResp, nil

          default:
            fmt.Printf("bundle %d unknown error: %s\n", i, bundleTxResp.bundleID)
          }
        }
      }
    }
    time.Sleep(poolInterval)
  }
  return bundleTxResp, errors.New("Couldn't get the bundle statuses: Max polling attempts reached. Final status uknown")
}

