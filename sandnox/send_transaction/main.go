package main
import(
  "context"
  "time"
  "log"
  "fmt"
  "os"
  "encoding/json"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/pumpdexer/programs/system"
  "github.com/scatkit/pumpdexer/rpc"
  "github.com/scatkit/gojito/jitorpc" 
)

func main() {
  solCl := rpc.New("https://api.mainnet-beta.solana.com")
	jitoCl := jitorpc.NewJito("https://frankfurt.mainnet.block-engine.jito.wtf", "") 
	
	defer solCl.Close()
	//defer jitoCl.Close()
  
  walletData, _ := os.ReadFile("wallet.json")
  var bts []byte
  if err := json.Unmarshal(walletData, &bts); err != nil{
    log.Fatal(err)
  }
  
  fromWallet := solana.PrivateKey(bts)
  toWallet := solana.MustPubkeyFromBase58("2gQov987LcCyHZq2BnLyKmkn9SG2i2F5hnfxPL7bymrS")
  transferAmount := uint64(95000)
  jitoTipAmount := uint64(5000)
  
  tipInst, err := jitoCl.GenerateJitoTipInstruction(context.Background(), jitoTipAmount, fromWallet.PublicKey())
	if err != nil {
		log.Fatal(err)
	}
  
  blockhash, err := solCl.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
  if err != nil {
    log.Fatal(err)
  }
  
  tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				transferAmount,
				fromWallet.PublicKey(),
				toWallet,
			).Build(),
			tipInst,
		},
		blockhash.Value.Blockhash,
		solana.TransactionPayer(fromWallet.PublicKey()),
	) 
  
  if err != nil{
    log.Fatal(err)
  }
  
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if fromWallet.PublicKey().Equals(key) {
			return &fromWallet
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	} 
  
  ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
  defer cancel()
  
  resp, err := jitoCl.SendTransaction(ctx, tx, true)
  if err != nil{
    log.Fatal(err)
  }
  
  fmt.Println(resp)
}



