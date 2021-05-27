package coin

type BlockHeader struct {
	Version float64
	BlockId string
	PrevBlockHash string
	MerkleRoot string
	Timestamp string
	Nonce int
	TargetBits float64
}


type Transaction struct {
	Amount float64
	ToAddress string
	FromAddress string
	Signature string
	PublicKey string
	Timestamp string
}


type Block struct {
	Hash string
	Header BlockHeader
	Body []Transaction
}


type RequestHeader struct {
	Node string  // wallet, node, miner
	Request string  // transaction, balalnce, dns, block mined, etc
}


type NetworkPacket struct {
	Header RequestHeader
	Body string  // serialised struct
}