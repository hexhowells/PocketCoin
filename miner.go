package main

import (
	"fmt"
	"pocketcoin/coin"
	"pocketcoin/blockchain"
	"pocketcoin/netpack"
	"math"
	"math/big"
	"time"
	"strconv"
	"strings"
	"net"
	"bufio"
	"encoding/json"
	"flag"
)

type BH = coin.BlockHeader
type TX = coin.Transaction
type BLK = coin.Block
type RQST = coin.RequestHeader

const (
	CONN_ADDR = "localhost"
	CONN_TYPE = "tcp"
)


var continueFlag = true
var transactionPool []coin.Transaction
var power = 231 // 233 = 2m12  |  231 = 4m39
var target = math.Pow(2, float64(power))

var nodeList = []string{"5555", "5556", "5557", "5558", "5559"}


func check(err error) {
	if err != nil {
		panic(err)
	}
}


func main() {

	argPortPtr := flag.String("p", "2222", "port to run the miner on")
	argWalletAddrPtr := flag.String("w", "", "miner's wallet address")
	argBlockchainFolderPtr := flag.String("f", "", "folder that stores the miners blockchain")
	flag.Parse()

	connPort := *argPortPtr
	walletAddress := *argWalletAddrPtr
	blockchainFolder := *argBlockchainFolderPtr

	if walletAddress == "" {
		fmt.Println("Missing command line argument [-w] - miner's wallet address")
		return
	}
	if blockchainFolder == "" {
		fmt.Println("Missing command line argument [-f] - folder that stores the miners blockchain")
		return
	}

	blockchain.SetBlockchainFolder(blockchainFolder)

	// check that the locally stored blockchain is valid
	fmt.Println("Checking blockchain...")
    blockchainValid, invalidBlock, invalidReason := blockchain.IsValid()
    if blockchainValid {
        fmt.Println("Blockchain valid, continuing...")
    } else {
        fmt.Printf("Blockchain found invalid on block %d\n", invalidBlock)
        fmt.Println("Reason block is invalid:", invalidReason)
        fmt.Println("Please repair blockchain\nExiting...")
        return
    }

	// check if the blockchain height matches the networks, if not then sync the blockchain
	localBlockHeight := blockchain.Height()
	bestNode, networkBlockHeight := getHighestNodeBlockHeight()
	if localBlockHeight != networkBlockHeight && networkBlockHeight != 0 {
		fmt.Println("\nBlockchain out of sync!")
		fmt.Printf("Local block height: %d  |  Network block height: %d\n", localBlockHeight, networkBlockHeight)
		heightDiff := (networkBlockHeight - localBlockHeight)
		fmt.Printf("Number of missing blocks: %d\n", heightDiff)
		fmt.Println("Syncing blockchain...")
		success := syncBlockchain(localBlockHeight, heightDiff, bestNode)
		if success {
			fmt.Println("Blockchain synced!\n")	
		} else {
			return
		}
		
	}

	go mineBlocks(walletAddress)

	ln, err := net.Listen(CONN_TYPE, CONN_ADDR + ":" + connPort)
	check(err)
	for {
		conn, err := ln.Accept()
		check(err)
		go handleConnection(conn)
	}
	
}


func mineBlocks(walletAddress string) {
	for {
		currentBlockHeight := blockchain.Height()
		blockId := strconv.Itoa(currentBlockHeight + 1)

		// build transaction body
		coinbase := constructCoinbaseTransaction(walletAddress)
		transactionBody := constructTransactionBody(coinbase)

		// generate merkle root
		transactionBodyString, _ := blockchain.Serialise(transactionBody)
		merkleRoot := blockchain.SHA256([]byte(transactionBodyString))

		// get prev block hash
		prevBlock := blockchain.GetHighestBlock()
		prevHeaderHash := prevBlock.Hash

		// construct block header
		blockHeader := constructBlockHeader(prevHeaderHash, blockId, merkleRoot)

		// mine block
		printStats(currentBlockHeight, len(transactionBody))
		fmt.Println("Mining Block...")
		start := time.Now()
		blockHash, blockTerminated := findHash(&blockHeader, target)
		fmt.Println("Hash time:", time.Since(start))

		// condition if another miner found the block
		if blockTerminated {
			newBlockHeight := blockchain.Height()
			newBlockFilename := "block_" + strconv.Itoa(newBlockHeight) + ".blk"
			newestBlockString, _ := blockchain.LoadBlock(newBlockFilename)
			newestBlock := blockchain.DeserialiseBlock(newestBlockString)

			refilTransactionPool(transactionBody[1:], newestBlock.Body)

			continue
		}

		// build and print the mined block
		block := constructBlock(blockHash, blockHeader, transactionBody)
		blockchain.PrettyPrint(block)

		// check validity of the new block
		blockValid, invalidReason := blockchain.VerifyBlock(block, prevBlock)
		if blockValid{
			// update blockchain
			serialisedBlock, _ := blockchain.Serialise(block)
			blockchain.Update(serialisedBlock, blockId)
			broadcastMinedBlock(block)
		} else {
			fmt.Println("Block Invalid:", invalidReason)
			break
		}
	}
}


func findHash(blockHeader *coin.BlockHeader, target float64) (string, bool) {
	nonce := 0
	blockHashString := ""
	b := big.NewFloat(target)
	terminated := false

	for continueFlag {
		blockHeader.Nonce = nonce
		hashString, _ := blockchain.Serialise(blockHeader)

		targetHashString := blockchain.SHA256([]byte(hashString))
		blockHashString = blockchain.SHA256([]byte(targetHashString))

		hashInt, _ := new(big.Int).SetString(blockHashString, 16)

		a := new(big.Float).SetInt(hashInt)
		
		if a.Cmp(b) == -1 {
			fmt.Printf("Block hash found after %d hashes\n", nonce)
			fmt.Println("Block hash:", blockHashString)
			break
		}
		nonce += 1
	}

	if continueFlag == false {
		fmt.Println("Mining stopped due to flag set")
		terminated = true
		continueFlag = true
	}

	return blockHashString, terminated
}


func handleConnection(conn net.Conn) {
	rawPacket, _ := bufio.NewReader(conn).ReadString('\n')

	packet := coin.NetworkPacket{}
	json.Unmarshal([]byte(rawPacket ), &packet)

	packetHeader := packet.Header

	if packetHeader.Request == "MinedBlock" {
		handleNewMinedBlock(packet)
	} else if packetHeader.Request == "Transaction" {
		handleNewTransaction(packet)
	}

}


func handleNewMinedBlock(packet coin.NetworkPacket) {
	newBlockString := packet.Body
	newBlock := blockchain.DeserialiseBlock(newBlockString)
	prevBlock := blockchain.GetHighestBlock()

	blockValid, invalidReason := blockchain.VerifyBlock(newBlock, prevBlock)
	if blockValid {
		fmt.Println("**New block valid!")
		continueFlag = false
		blockId := blockchain.Height() + 1
		_ = blockchain.Update(newBlockString, strconv.Itoa(blockId))
	} else {
		fmt.Println("**New block found not valid!")
		fmt.Println("**Reason:", invalidReason)
	}
}


func handleNewTransaction(packet coin.NetworkPacket) {
	newTxString := packet.Body
	newTx := blockchain.DeserialiseTransaction(newTxString)

	if !transactionInList(newTx, transactionPool) {
		transactionPool = append(transactionPool, newTx)
	}

}


func printStats(currentBlockHeight int, numTxInBlock int) {
	fmt.Print("\n")
	fmt.Println(strings.Repeat("#", 50))
	fmt.Printf("Current Block Height: %d\n", currentBlockHeight)
	fmt.Printf("Number of transactions in pool: %d\n", len(transactionPool))
	fmt.Printf("Number of transactions in current block: %d\n", numTxInBlock)
}


func constructBlockHeader(prev_block_hash string, blockId string, merkleRoot string) coin.BlockHeader{
	bHeader := BH{}

	bHeader.Version = 0.1
	bHeader.BlockId = blockId
	bHeader.PrevBlockHash = prev_block_hash
	bHeader.MerkleRoot = merkleRoot
	bHeader.Timestamp = time.Now().String()
	bHeader.TargetBits = target

	return bHeader
}


func constructTransactionBody(coinbaseTransaction coin.Transaction) []coin.Transaction {
	txBody := []TX{}
	txBody = append(txBody, coinbaseTransaction)
	var tx coin.Transaction

	lim := len(transactionPool)
	for i := 0; i < lim; i++ {
		if i >= 10 {
			break
		}
		tx, transactionPool = transactionPool[0], transactionPool[1:]
		txBody = append(txBody, tx)
	}

	return txBody
}


func refilTransactionPool(transactions []coin.Transaction, txList []coin.Transaction) {
	for _, tx := range transactions {
		if transactionInList(tx, txList) {
			continue
		}
		transactionPool = append([]coin.Transaction{tx}, transactionPool...)
	}
}


func transactionInList(tx coin.Transaction, blockBody []coin.Transaction) bool {
	for _, blockTx := range blockBody {
		if tx == blockTx {
			return true
		}
	}
	return false
}


func constructCoinbaseTransaction(walletAddress string) coin.Transaction {
 	coinbase := TX{}
 	coinbase.Amount = 10.0
 	coinbase.ToAddress = walletAddress
 	coinbase.FromAddress = "coinbase"
 	coinbase.Signature = ""
 	coinbase.PublicKey = ""
 	coinbase.Timestamp = time.Now().String()

 	return coinbase
} 


func getMerkleRoot(transactions []coin.Transaction) string {
	transactionBodyString, _ := blockchain.Serialise(transactions)
	transactionBodyHash := blockchain.SHA256([]byte(transactionBodyString))

	return transactionBodyHash
}


func hashPrevBlockHeader(prevBlock coin.Block) string {
	prevHeaderString, _ := blockchain.Serialise(prevBlock.Header)
	prevHeaderHash := blockchain.SHA256([]byte(prevHeaderString))

	return prevHeaderHash
}


func constructBlock(hash string, header coin.BlockHeader, body []coin.Transaction) coin.Block {
	block := BLK{}

	block.Hash = hash
	block.Header = header
	block.Body = body

	return block
}



func broadcastMinedBlock(block coin.Block) {
	blockString, _ := blockchain.Serialise(block)
	reqHeader := netpack.ConstructRequestHeader("miner", "MinedBlock")
	packet := netpack.ConstructNetworkPacket(reqHeader, blockString)

	for _, addr := range nodeList {
		packetString, _ := blockchain.Serialise(packet)
		netpack.BroadcastPacket(packetString, addr)
	}

}


func addToTransactionPool(tx coin.Transaction) {
	transactionPool = append(transactionPool, tx)
}


func getHighestNodeBlockHeight() (string, int) {
	highest := 0
	bestNode := ""

	for _, port := range nodeList {
		height := getNetworkBlockHeight(port)
		if height >  highest {
			highest = height
			bestNode = port
		}
	}

	return bestNode, highest
}


func getNetworkBlockHeight(port string) int {
	reqHeader := netpack.ConstructRequestHeader("miner", "BlockHeight")
	packet := netpack.ConstructNetworkPacket(reqHeader, "")
	packetString, _ := blockchain.Serialise(packet)
	success, response := netpack.BroadcastDuplexPacket(packetString, port)

	if success {
		networkHeight, _ := strconv.Atoi(response.Body)
		return networkHeight
	} else {
		return -1
	}
}


func syncBlockchain(blockHeight int, heightDiff int, port string) bool {
	prevBlock := blockchain.GetHighestBlock()

	// establish connection with node
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		fmt.Println("Unable to establish a connection with the specified node")
		return false
	}

	// create blockchain sync initialisation packet
	reqHeader := netpack.ConstructRequestHeader("miner", "SyncBlockchain")
	packet := netpack.ConstructNetworkPacket(reqHeader, strconv.Itoa(blockHeight))
	packetString, _ := blockchain.Serialise(packet)

	// send blockchain sync initialisation request
	fmt.Fprintf(conn, packetString + "\n")

	// check if node can start the syncing routine
	recv, _ := bufio.NewReader(conn).ReadString('\n')
	if recv[:len(recv)-1] != "Okay" {
		fmt.Println("Node unable to start syncing routine")
		return false
	}

	// begin syncing the blockchain
	for i:=0; i < heightDiff; i++ {
		blockStringRaw, _ := bufio.NewReader(conn).ReadString('\n')
		blockString := blockStringRaw[:len(blockStringRaw)-1]
		block := blockchain.DeserialiseBlock(blockString)

		blockValid, invalidReason := blockchain.VerifyBlock(block, prevBlock)

		if blockValid {
			blockId := strconv.Itoa((blockHeight+i)+1)
			blockchain.Update(blockString, blockId)
			prevBlock = block	
		} else {
			fmt.Printf("Block %d from network invalid!\n", (blockHeight+i))
			fmt.Println("Invalid reason: ", invalidReason)
			return false
		}

		fmt.Fprintf(conn, "Okay\n")
	}

	conn.Close()

	return true
}