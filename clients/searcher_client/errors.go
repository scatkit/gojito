package searcher_client
import "fmt"

type BundleRejectionError struct {
	Message string
}

func (e BundleRejectionError) Error() string {
	return e.Message
}

func NewStateAuctionBidRejectedError(auction string, tip uint64) error {
	return BundleRejectionError{
		Message: fmt.Sprintf("bundle lost state auction, auction: %s, tip %d lamports", auction, tip),
	}
}

func NewWinningBatchBidRejectedError(auction string, tip uint64) error {
	return BundleRejectionError{
		Message: fmt.Sprintf("bundle won state auction but failed global auction, auction %s, tip %d lamports", auction, tip),
	}
}

func NewSimulationFailureError(tx string, message string) error {
	return BundleRejectionError{
		Message: fmt.Sprintf("bundle simulation failure on tx %s, message: %s", tx, message),
	}
}

func NewInternalError(message string) error {
	return BundleRejectionError{
		Message: fmt.Sprintf("internal error %s", message),
	}
}

func NewDroppedBundle(message string) error {
	return BundleRejectionError{
		Message: fmt.Sprintf("bundle dropped %s", message),
	}
}

//type BundleRejectionError struct{
//  Message string
//}
//
//type BundleRejection interface{
//  setErr(msg *BundleRejectionError)
//}
//
//func (e BundleRejectionError) Error() string{ 
//   return e.Message
//}
//
//
//type BundleRejectionErrorFunc func(err *BundleRejectionError) 
//
//func (f BundleRejectionErrorFunc) setErr(err *BundleRejectionError){
//  f(err)
//}
//
//func NewSimulationFailureError(tx string, message string) BundleRejection{
//  return BundleRejectionErrorFunc(func(err *BundleRejectionError){
//    err.Message = fmt.Sprintf("bundle simulation failure on tx %s, message: %s", tx, message)
//  })
//}
// transactionOptionsFields (holds fileds)

// TO -> interface for TOF

// TOF is of type func(opts *transactionOptionsFields) 
// TOF has method apply(option) which runs TOF with the option

// TransPayer(payer PublicKey) TransactionOption{ 
//  return func(opts *transactionOptionsFields){ opts.payer = payer} of type TOF
// }
// 
// opts := [TransPayer,]
// some_options := TransactionOptionsFiled{payer: PublicKey}
// for opt in opts -> opt.apply(&some_options) <- it mutates the fileds
