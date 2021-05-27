package main

import (
    "fmt"
    "net"
    "bufio"
    "strconv"
    "flag"
    b64 "encoding/base64"
    "encoding/json"
    "io/ioutil"

    "pocketcoin/coin"
    "pocketcoin/blockchain"
    "pocketcoin/netpack"
    "pocketcoin/pgp"
)

type T = coin.Transaction
type H = coin.RequestHeader

var transactionPool []coin.Transaction
var pgpCache []PGPCacheEntry



// ---- PGP Public Key Caching Solution ----
// instead of including the wallets pgp key every transaction
// only include it for a wallets first transaction
// if a pgp key is included, in a seperate file store the wallet address and blockId where that wallets pgp public key is stored
// public key is then secured by the blockchain without having to search throught the entire blockchain to find the key
// only including the key in the wallets first transaction allows for smaller file sizes



const (
    CONN_ADDR = "localhost"
    CONN_PORT = "5555"
    CONN_TYPE = "tcp"
)


type PGPCacheEntry struct {
    WalletAddress string
    PublicKeyPem string
}


var minerPortList = []string{"2221", "2222", "2223", "2224", "2225"}
var nodeList = []string{"5555", "5556", "5557", "5558", "5559"}


func check(err error) {
    if err != nil {
        panic(err)
    }
}


func main() {
    argBlockchainFolderPtr := flag.String("f", "", "folder that stores the nodes blockchain")
    argPortPtr := flag.String("p", "5555", "port that the node listens on")
    flag.Parse()

    blockchainFolder := *argBlockchainFolderPtr
    port := *argPortPtr

    if blockchainFolder == "" {
        fmt.Println("Missing command line argument [-f] - folder that stores the nodes blockchain")
        return
    }

    blockchain.SetBlockchainFolder(blockchainFolder)

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
    bestNode, networkBlockHeight := blockchain.GetHighestNodeBlockHeight(nodeList)
    if localBlockHeight != networkBlockHeight && networkBlockHeight != 0 {
        fmt.Println("\nBlockchain out of sync!")
        fmt.Printf("Local block height: %d  |  Network block height: %d\n", localBlockHeight, networkBlockHeight)
        heightDiff := (networkBlockHeight - localBlockHeight)
        fmt.Printf("Number of missing blocks: %d\n", heightDiff)
        fmt.Println("Syncing blockchain...")
        success := blockchain.SyncNodeBlockchain(localBlockHeight, heightDiff, bestNode)
        if success {
            fmt.Println("Blockchain synced!\n") 
        } else {
            return
        }
    }

    loadPGPCache()
    
    fmt.Println("listening on", CONN_ADDR + ":" + port);
    ln, err := net.Listen(CONN_TYPE, CONN_ADDR + ":" + port)
    check(err)
    for {
        conn, err := ln.Accept() // this blocks until connection or error
        check(err)
        go handleConnection(conn) // a goroutine handles conn so that the loop can accept other connections
    }
}


func handleConnection(conn net.Conn) {
    fmt.Println("New Connection From:", conn.RemoteAddr().String())

    rawPacket, _ := bufio.NewReader(conn).ReadString('\n')
    packet := netpack.DeserialisePacket(rawPacket)
    head := packet.Header

    blockchain.PrettyPrint(packet)

    // process the body correctly
    switch head.Request {
    case "Transaction":
        handleTransaction(packet.Body)
    case "Balance":
        responsePacket := handleBalanceRequest(packet.Body)
        conn.Write([]byte(responsePacket))
    case "MinedBlock":
        handleBlockMined(packet.Body)
    case "BlockHeight":
        responsePacket := handleBlockHeight()
        conn.Write([]byte(responsePacket))
    case "SyncBlockchain":
        syncBlockchain(conn, packet.Body)
    case "PublicKeyInCache":
        responsePacket := handlePublicKeyInCache(packet.Body)
        conn.Write([]byte(responsePacket))
    }

    conn.Close()
}


func handleTransaction(bodyString string) {
    tx := blockchain.DeserialiseTransaction(bodyString)
    if transactionValid(tx) {
        if !publicKeyInCache(tx.FromAddress) {
            addToPgpCache(tx.FromAddress, tx.PublicKey)
        }
        transactionPool = append(transactionPool, tx)
        broadcastTransaction(bodyString)
        fmt.Printf("Number of transactions in pool: %d\n", len(transactionPool))
    } else {
        fmt.Println("Recieved transaction invalid!")
    }
    
    blockchain.PrettyPrint(tx)
}


func handleBalanceRequest(bodyString string) string {
    walletAddr := bodyString
    // verify wallet address is valid
    balance := getWalletBalanceWithPool(walletAddr)
    balanceString := fmt.Sprintf("%f", balance)

    respHeader := netpack.ConstructRequestHeader("node", "Response")
    respPacket := netpack.ConstructNetworkPacket(respHeader, balanceString)
    packetString, _ := blockchain.Serialise(respPacket)

    return packetString
}


func handleBlockMined(newBlockString string) {
    newBlock := blockchain.DeserialiseBlock(newBlockString)
    prevBlock := blockchain.GetHighestBlock()
    blockValid, invalidReason := blockchain.VerifyBlock(newBlock, prevBlock)

    if blockValid {
        blockId := strconv.Itoa(blockchain.Height() + 1)
        blockchain.Update(newBlockString, blockId)
        fmt.Println("New Block Mined!")
        broadcastNewBlock(newBlockString)
        updateTransactionPool(newBlock)
    } else {
        fmt.Println("Block invalid. Reason:", invalidReason)
    }
    
}


func broadcastNewBlock(newBlockString string) {
    reqHeader := netpack.ConstructRequestHeader("node", "MinedBlock")
    packet := netpack.ConstructNetworkPacket(reqHeader, newBlockString)
    packetString, _ := blockchain.Serialise(packet)
    for _, addr := range minerPortList {
        netpack.BroadcastPacket(packetString, addr)
    }
    for _, addr := range nodeList {
        netpack.BroadcastPacket(packetString, addr)
    }
}


func handleBlockHeight() string {
    blockHeight := blockchain.Height()
    respHeader := netpack.ConstructRequestHeader("node", "Response")
    respPacket := netpack.ConstructNetworkPacket(respHeader, strconv.Itoa(blockHeight))
    packetString, _ := blockchain.Serialise(respPacket)

    return packetString
}


func handlePublicKeyInCache(walletAddress string) string {
    var inCache string
    if publicKeyInCache(walletAddress) {
        inCache = "true"
    } else {
        inCache = "false"
    }

    respHeader := netpack.ConstructRequestHeader("node", "PublicKeyInCache")
    respPacket := netpack.ConstructNetworkPacket(respHeader, inCache)
    packetString, _ := blockchain.Serialise(respPacket)

    return packetString
}


func syncBlockchain(conn net.Conn, blockHeightString string) {
    conn.Write([]byte("Okay\n"))
    minerBlockHeight, _ := strconv.Atoi(blockHeightString)
    blockHeight := blockchain.Height()


    for i:=minerBlockHeight+1; i <= blockHeight; i++ {
        filename := "block_" + strconv.Itoa(i) + ".blk"
        blockString, _ := blockchain.LoadBlock(filename)
        conn.Write([]byte(blockString + "\n"))

        blockSuccess, _ := bufio.NewReader(conn).ReadString('\n')
        if blockSuccess != "Okay\n" {
            break
        }
    }
}


func broadcastTransaction(txString string) {
    reqHeader := netpack.ConstructRequestHeader("node", "Transaction")
    packet := netpack.ConstructNetworkPacket(reqHeader, txString)
    packetString, _ := blockchain.Serialise(packet)
    for _, addr := range minerPortList {
        netpack.BroadcastPacket(packetString, addr)
    }
    for _, addr := range nodeList {
        netpack.BroadcastPacket(packetString, addr)
    }
}


func transactionValid(tx coin.Transaction) bool {
    balance := getWalletBalanceWithPool(tx.FromAddress)
    publicKeyPem, publicKeyExists := getWalletPublicKeyPem(tx)
    var valid bool
    var publicKeyHash string

    if publicKeyExists {
        publicKeyHash = blockchain.SHA256([]byte(publicKeyPem))[:32]
    }

    if tx.Amount <= 0 || tx.Amount > balance {
        valid = false
    } else if tx.FromAddress == tx.ToAddress {
        valid = false
    } else if transactionInList(tx, transactionPool) {
        valid = false
    } else if !publicKeyExists {
        valid = false
    } else if !transactionSignatureValid(tx, publicKeyPem) {
        valid = false
    } else if tx.FromAddress != publicKeyHash {
        valid = false
    } else {
        valid = true
    }

    return valid
}


func transactionSignatureValid(tx coin.Transaction, publicKeyPem string) bool {
    signatureString := tx.Signature
    tx.Signature = ""

    publicKey, _ := pgp.ParsePublicKeyFromPemStr(publicKeyPem)
    txString, _ := blockchain.Serialise(tx)
    signature, _ := b64.StdEncoding.DecodeString(signatureString)

    valid := pgp.ValidSignature(txString, signature, publicKey)

    return valid
}


func getWalletBalance(wallet string) float64 {
    networkHeight := blockchain.Height()
    balance := 0.0

    for i:=0; i <= networkHeight; i++ {
        filename := "block_" + strconv.Itoa(i) + ".blk"
        blockString, _ := blockchain.LoadBlock(filename)
        block := blockchain.DeserialiseBlock(blockString)

        for _, tx := range block.Body {
            if tx.ToAddress == wallet {
                balance += tx.Amount
            } else  if tx.FromAddress == wallet {
                balance -= tx.Amount
            }
        }
    }
    return balance
}


func getWalletBalanceWithPool(wallet string) float64 {
    balance := getWalletBalance(wallet)

    for _, tx := range transactionPool {
        if tx.FromAddress == wallet {
            balance -= tx.Amount
        }
    }

    return balance
}


func transactionInList(tx coin.Transaction, blockBody []coin.Transaction) bool {
    for _, blockTx := range blockBody {
        if tx == blockTx {
            return true
        }
    }
    return false
}


func updateTransactionPool(block coin.Block) {
    for _, tx := range block.Body[1:] {
        txIndex := indexInTxPool(transactionPool, tx)
        if txIndex != -1 {
            transactionPool = append(transactionPool[:txIndex], transactionPool[txIndex+1:]...)
        }
    }
}


func indexInTxPool(list []coin.Transaction, item coin.Transaction) int {
    for i, elm := range list {
        if elm == item {
            return i
        }
    }
    return -1
}


func loadPGPCache() {
    cacheString, _ := ioutil.ReadFile("NodeCache/pgpCache.txt")
    json.Unmarshal([]byte(cacheString), &pgpCache)
}


func addToPgpCache(walletAddress string, PublicKeyPem string) {
    cacheEntry := PGPCacheEntry{}
    cacheEntry.WalletAddress = walletAddress
    cacheEntry.PublicKeyPem = PublicKeyPem

    pgpCache = append(pgpCache, cacheEntry)
    savePgpCacheFile()

}


func savePgpCacheFile() {
    cacheString, _ := blockchain.Serialise(pgpCache)
    _ = ioutil.WriteFile("NodeCache/pgpCache.txt", []byte(cacheString), 0644)
}


func publicKeyInCache(walletAddress string) bool {
    for _, txCacheEntry := range pgpCache {
        if walletAddress == txCacheEntry.WalletAddress {
            return true
        }
    }
    return false
}


func getWalletPublicKeyPem(tx coin.Transaction) (string, bool) {
    if tx.PublicKey != "" {
        return tx.PublicKey, true
    } else if publicKeyInCache(tx.FromAddress) {
        return getPublicKeyFromCache(tx.FromAddress), true
    } else {
        return "", false
    }
}


func getPublicKeyFromCache(walletAddress string) string {
    for _, txCacheEntry := range pgpCache {
        if walletAddress == txCacheEntry.WalletAddress {
            return txCacheEntry.PublicKeyPem
        }
    }
    return ""
}

