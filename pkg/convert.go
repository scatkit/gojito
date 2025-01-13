package pkg
import(
  "github.com/scatkit/pumpdexer/solana"
  "github.com/scatkit/gojito/pb"
)

// Converts a pumpdexer's `solana.Transaction` to a pb.Packet
func ConvertTransactionToProtobufPacket(transaction *solana.Transaction) (jito_pb.Packet, error){
  tx_data, err := transaction.MarshalBinary()
  if err != nil{
    return jito_pb.Packet{}, err
  }
  
  return jito_pb.Packet{
    Data: tx_data,
    Meta: &jito_pb.Meta{
      Size: uint64(len(tx_data)),
      Addr: "",
      Flags: nil,
      SenderStake: 0,
    },
  }, nil
}
