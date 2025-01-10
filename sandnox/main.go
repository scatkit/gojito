package main
import (
  "context"
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/pumpdexer/rpc"
  "github.com/scatkit/gojito/clients/searcher_client"
  "github.com/scatkit/pumpdexer/programs/system"
  "log"
  "time"
)

func main(){
  privKeyBase58:= "5rg7jXrAYXoAYt1ARV1RzuRFCsH948MyjMjKVG8Kiw7pdZZ7QBjuJnEfufvukPJ5hLyRHUXkPBuc9mP7AS35i5yC"
  
  key, err := solana.PrivateKeyFromBase58(privKeyBase58)
  if err != nil{
    log.Fatal(err)
  }
  
  // Creating a searcher client
  client, err := searcher_client.New(
    context.Background(),
    "ny.mainnet.block-engine.jito.wtf:443", // grpcDialUrl
    rpc.New("https://mainnet.block-engine.jito.wtf/api/v1"), // jitoRPCClient
    rpc.New("https://api.mainnet-beta.solana.com"), // solana's rpc
    key, // private key
    nil, // no opts
  ) 
  if err != nil{
    log.Fatal(err)
  }
  defer client.Close()
  
  ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
  defer cancel() 
  
  fundedWallet, err := solana.PrivateKeyFromBase58(privKeyBase58)
  if err != nil{
    log.Fatal(err)
  }
  
  fundedWalletPubKey := fundedWallet.PublicKey()

  blockHash, err := client.RpcConn.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
  if err != nil{
    log.Fatal(err)
  } 
  
  tipInstr, err := client.GenerateTipRandomAccountInstruction(1000000, fundedWalletPubKey)
  
  tx, err := solana.NewTransaction(
    []solana.Instruction{
      system.NewTransferInstruction(
        10000000, // lamports
        fundedWallet.PublicKey(), // sender
        solana.MustPubkeyFromBase58("A6njahNqC6qKde6YtbHdr1MZsB5KY9aKfzTY1cj8jU3v"), // receiver
      ).Build(),
      tipInstr,
    },
    blockHash.Value.Blockhash,
    solana.TransactionPayer(fundedWalletPubKey),
  )
  
  if err != nil{
    log.Fatal(err)
  }

  _,err = tx.Sign(
    func(key solana.PublicKey) *solana.PrivateKey{
      if fundedWalletPubKey.Equals(key){
        return &fundedWallet
      }
      return nil
    },
  )
  
  if err != nil{
    log.Fatal(err)
  }
  
  resp, err := client.SimulateBundle(
    ctx,
    searcher_client.SimulateBundleParams{
      EncodedTransactions: []string{tx.MustToBase64()},
    },
    searcher_client.SimulateBundleConfig{
      PreExecutionAccountsConfigs: []searcher_client.ExecutionAccounts{
				{
					Encoding:  "base64",
					Addresses: []string{"3vjULHsUbX4J2nXZJQQSHkTHoBqhedvHQPDNaAgT9dwG"},
				},
			},
      PostExecutionAccountsConfigs: []searcher_client.ExecutionAccounts{
				{
					Encoding:  "base64",
					Addresses: []string{"3vjULHsUbX4J2nXZJQQSHkTHoBqhedvHQPDNaAgT9dwG"},
				},
			},
    },
  )
  
  if err != nil {
		log.Fatal(err)
	}
  
  log.Println(resp)
}
