package main
import (
  "context"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/davecgh/go-spew/spew"
  "github.com/scatkit/pumpdexer/rpc"
  "github.com/scatkit/gojito/clients/searcher_client"
  "github.com/scatkit/pumpdexer/programs/system"
  "log"
  "time"
)

func main(){
  
  // Creating a searcher client
  client, err := searcher_client.NewNoAuth(
    context.Background(),
    "ny.mainnet.block-engine.jito.wtf:443", // grpcDialUrl
    rpc.New("https://mainnet.block-engine.jito.wtf/api/v1"), // jitoRPCClient
    rpc.New("https://api.mainnet-beta.solana.com"), // solana's rpc
    nil, // tls.config
    "", // proxy
    //"IP:PORT:USERNAME:PASSWORD", // this is a placeholder value showcasing the proxy format you should use, but you may have this field empty
  ) 
  if err != nil{
    log.Fatal(err)
  }
  defer client.Close()
  
  ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
  defer cancel() 

   // Max 5 transactions per bundle
  txns := make([]*solana.Transaction, 0, 5)
  blockHash, err := client.RpcConn.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
  if err != nil{
    log.Fatal(err)
  } 
  
  fromWallet := solana.MustPrivkeyFromBase58("5rg7jXrAYXoAYt1ARV1RzuRFCsH948MyjMjKVG8Kiw7pdZZ7QBjuJnEfufvukPJ5hLyRHUXkPBuc9mP7AS35i5yC")
	toWallet := solana.MustPubkeyFromBase58("BLrQPbKruZgFkNhpdGGrJcZdt1HnfrBLojLYYgnrwNrz")
  
  // Generates a tip instruction sent to 1 of 8 random Jito Tip wallets
  tipInstr, err := client.GenerateTipRandomAccountInstruction(1000000, fromWallet.PublicKey())
  if err != nil{
    log.Fatal(err)
  }
  
  tx, err := solana.NewTransaction(
    []solana.Instruction{
      system.NewTransferInstruction(
        10000000, // lamports
        fromWallet.PublicKey(), // sender
        toWallet,
      ).Build(),
      tipInstr,
    },
    blockHash.Value.Blockhash,
    solana.TransactionPayer(fromWallet.PublicKey()),
  )
  
  if err != nil{
    log.Fatal(err)
  }

  _,err = tx.Sign(
    func(key solana.PublicKey) *solana.PrivateKey{
      if fromWallet.PublicKey().Equals(key){
        return &fromWallet
      }
      return nil
    },
  )
  
  if err != nil{
    log.Fatal(err)
  }
  
  spew.Dump(tx)
  
  txns = append(txns, tx)
  
  resp, err := client.BroadcastBundleWithConfirmation(ctx, txns)
	if err != nil {
		log.Fatal(err)
	}
  
  log.Println(resp)
}
