package blockchain

import (
	"io/ioutil"
	"bufio"
	"pocketcoin/coin"
	"pocketcoin/netpack"
	"strconv"
	"net"
	"encoding/json"
	"fmt"
	"crypto/sha256"
)

var blockchainFolder string = "Blockchain"


func SetBlockchainFolder(folder string) {
	blockchainFolder = folder
}


func Height() int {
	block_files, _ := ioutil.ReadDir(blockchainFolder)
	return len(block_files) - 1
}


func GetHighestBlock() coin.Block {
	currentBlockHeight := Height()
	blockFilename := "block_" + strconv.Itoa(currentBlockHeight) + ".blk"
	blockString, _ := LoadBlock(blockFilename)
	block := DeserialiseBlock(blockString)

	return block
}


func LoadBlock(blockFilename string) (string, error) {
	filepath := blockchainFolder + "/" + string(blockFilename)
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}


func Update(serialisedBlock string, block_id string) error {
	block_bytes := []byte(serialisedBlock)
	filepath := blockchainFolder + "/block_" + block_id + ".blk"
	err := ioutil.WriteFile(filepath, block_bytes, 0644)

	return err
}


func SHA256(byteData []byte) string {
	hashHex := sha256.Sum256(byteData)
	hashString := fmt.Sprintf("%x", hashHex)
	return hashString
}


func VerifyBlock(block coin.Block, prevBlock coin.Block) (bool, string) {
	// check block hash has correct number of leading zeros
	blockHashLeading := block.Hash[0:6]
	if blockHashLeading != "000000"{
		return false, "Block hash leading zeros invalid"
	}

	// check the previous block hash
	prevBlockHeaderHash := prevBlock.Hash
	if prevBlockHeaderHash != block.Header.PrevBlockHash {
		return false, "Previous block hash invalid"
	}

	// Check the transaction body hash
	blockBodyString, _ := Serialise(block.Body)
	blockBodyHash := SHA256([]byte(blockBodyString))
	if blockBodyHash != block.Header.MerkleRoot {
		return false, "Merkle root hash invalid"
	}

	// Check the block hash
	blockHeaderString, _ := Serialise(block.Header)
	firstHash := SHA256([]byte(blockHeaderString))
	checkHash := SHA256([]byte(firstHash))
	if block.Hash != checkHash {
		return false, "Block hash invalid"
	}

	// check coinbase transaction is the specified amount
	if block.Body[0].Amount != 10{
		return false, "Coinbase transaction amount invalid"
	}

	return true, ""
}



func DeserialiseBlock(blockString string) coin.Block {
	block := coin.Block{}
	json.Unmarshal([]byte(blockString), &block)

	return block
}


func DeserialiseBlockHeader(blockHeaderString string) coin.BlockHeader {
	block := coin.BlockHeader{}
	json.Unmarshal([]byte(blockHeaderString), &block)

	return block
}


func DeserialiseTransaction(transactionString string) coin.Transaction {
	tx := coin.Transaction{}
	json.Unmarshal([]byte(transactionString), &tx)

	return tx
}


func Serialise(data interface{}) (string , error) {
	out, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(out), nil
}


func PrettyPrint(data interface{}) {
	var p []byte
	p, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s \n", p)
}


func IsValid() (bool, int, string) {
	prevBlockString, _ := LoadBlock("block_0.blk")
	prevBlock := DeserialiseBlock(prevBlockString)
	height := Height()

	for i:=1; i <= height; i++ {
		filename := "block_" + strconv.Itoa(i) + ".blk"
		blockString, _ := LoadBlock(filename)
		block := DeserialiseBlock(blockString)
		valid, invalidReason := VerifyBlock(block, prevBlock)
		if valid {
			prevBlock = block
		} else {
			return false, i, invalidReason
		}
	}
	return true, -1, ""
}


// Below are functions for syncing the blockchain, used by the nodes and miners
func GetHighestNodeBlockHeight(nodeList []string) (string, int) {
    highest := 0
    bestNode := ""

    for _, port := range nodeList {
        height := GetNetworkBlockHeight(port)
        if height >  highest {
            highest = height
            bestNode = port
        }
    }

    return bestNode, highest
}


func GetNetworkBlockHeight(port string) int {
    reqHeader := netpack.ConstructRequestHeader("generic", "BlockHeight")
    packet := netpack.ConstructNetworkPacket(reqHeader, "")
    packetString, _ := Serialise(packet)
    success, response := netpack.BroadcastDuplexPacket(packetString, port)

    if success {
        networkHeight, _ := strconv.Atoi(response.Body)
        return networkHeight
    } else {
        return -1
    }
}


func SyncNodeBlockchain(blockHeight int, heightDiff int, port string) bool {
    prevBlock := GetHighestBlock()

    // establish connection with node
    conn, err := net.Dial("tcp", "localhost:"+port)
    if err != nil {
        fmt.Println("Unable to establish a connection with the specified node")
        return false
    }

    // create blockchain sync initialisation packet
    reqHeader := netpack.ConstructRequestHeader("generic", "SyncBlockchain")
    packet := netpack.ConstructNetworkPacket(reqHeader, strconv.Itoa(blockHeight))
    packetString, _ := Serialise(packet)

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
        block := DeserialiseBlock(blockString)

        blockValid, invalidReason := VerifyBlock(block, prevBlock)

        if blockValid {
            blockId := strconv.Itoa((blockHeight+i)+1)
            Update(blockString, blockId)
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