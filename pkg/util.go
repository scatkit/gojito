package pkg
import(
  "github.com/scatkit/pumpdexer/solana"
)

// NewKeyPair creates a Keypair from a private key.
func NewKeyPair(privateKey solana.PrivateKey) *Keypair {
	return &Keypair{PrivKey: privateKey, PubKey: privateKey.PublicKey()}
}

func BatchExtractSigFromTx(txns []*solana.Transaction) []solana.Signature{
  // The first transaction is singed by the first public key
  sigs := make([]solana.Signature, 0, len(txns))
  for _, tx := range txns{
    sigs = append(sigs, tx.Signatures[0])
  }
  return sigs
}
