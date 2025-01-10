package pkg
import(
  "github.com/scatkit/pumpdexer/solana"
)

// NewKeyPair creates a Keypair from a private key.
func NewKeyPair(privateKey solana.PrivateKey) *Keypair {
	return &Keypair{PrivKey: privateKey, PubKey: privateKey.PublicKey()}
}
