package main

import (
	"fmt"
	"flag"
	"io/ioutil"
	"os"
	"bufio"
	"strings"
	"strconv"
	"errors"
	"time"

	"pocketcoin/coin"
	"pocketcoin/pgp"
	"pocketcoin/blockchain"
	"pocketcoin/netpack"

    "crypto/sha256"
    "encoding/hex"
    b64 "encoding/base64"
)


var walletFilepath string
var nodeList = []string{"5555", "5556", "5557", "5558", "5559"}


func check(err error) {
	if err != nil {
		panic(err)
	}
}


func main() {
	walletFilepathPtr := flag.String("f", "", "folder for wallet")
	balancePtr := flag.Bool("b", false, "display wallet balance")
	transactionPtr := flag.Bool("t", false, "Send a transaction")
	addrPtr := flag.Bool("w", false, "show wallet address")
	newAddrPtr := flag.Bool("n", false, "create a new wallet address")
	flag.Parse()

	balanceFlag := *balancePtr
	addrFlag := *addrPtr
	transactionFlag := *transactionPtr
	newAddrFlag := *newAddrPtr
	walletFilepath = *walletFilepathPtr

	if walletFilepath == "" {
		fmt.Println("Missing command line argument [-f] - folder name of the wallet")
		fmt.Println("This argument is required for shards!")
		return
	}

	if balanceFlag {
		walletAddress := loadWalletAddress()
		balance := requestWalletBalance(walletAddress)
		fmt.Println("Wallet balance:", balance)
	}

	if addrFlag {
		walletAddress := loadWalletAddress()
		fmt.Println("Wallet address:", walletAddress)
	}

	if transactionFlag {
		// get the address to send the coins to
		fmt.Print("Wallet Address to send coins to: ")
		toAddr, err := getSendAddress()
		check(err)
		
		// get the amount of coins to send to the address
		fmt.Print("Amount to send (up to 6 decimal points): ")
		amount := getAmountToSend()
		
		// get the sender address from memory
		fromAddr := loadWalletAddress()

		fmt.Printf("\nSending %f to %s from %s\n", amount, toAddr, fromAddr)

		transactionPacket := constructTransactionPacket(toAddr, fromAddr, amount)
		broadcastTransactionToNetwork(transactionPacket)
		fmt.Println("\nTransaction successfully sent!")


		// connect to a node in the network
		// transmit transaction struct to the node, signed with private key, also send the public key
		// node hashes the public key to verify the wallets public key, then use that to verify the signature
		// handle any errors returned back from the node
	}

	if newAddrFlag {
		fmt.Print("Creating a new wallet will delete the previously stored wallet, are you sure? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		check(err)

		if strip(confirmation) == "y" {
			walletAddress := generateWalletAddress()
			fmt.Println("New wallet created!")
			fmt.Println("Wallet address:", walletAddress)
		} else {
			fmt.Println("Canceling operation.")
			return
		}
	}
}


func generateWalletAddress() string {
	// generate new private-public key pair
	privateKey, publicKey := pgp.GenerateKeyPair()

	// Export the keys to pem string
    privPem := pgp.ExportPrivateKeyAsPemStr(privateKey)
    pubPem, _ := pgp.ExportPublicKeyAsPemStr(publicKey)

	// save the public and private key
	saveKey(string(pubPem), walletFilepath+"pub.asc")
	saveKey(string(privPem), walletFilepath+"priv.asc")

	// hash the public key
	publicKeyHash := sha256.Sum256([]byte(pubPem))
	walletAddress := hex.EncodeToString(publicKeyHash[:16]) // truncate address

	saveKey(string(walletAddress), walletFilepath+"walletAddress.txt")

	return walletAddress
}


func saveKey(key string, filename string) {
	keyBytes := []byte(key)
	err := ioutil.WriteFile(filename, keyBytes, 0644) // last param - filemode
	check(err)
}


func loadWalletAddress() string {
	walletAddress, err := ioutil.ReadFile(walletFilepath+"walletAddress.txt")
	check(err)
	return string(walletAddress)
}


func loadPrivateKeyPem() string {
	privateKeyPem, err := ioutil.ReadFile(walletFilepath+"priv.asc")
	check(err)
	return string(privateKeyPem)
}


func loadPublicKeyPem() string {
	publicKeyPem, err := ioutil.ReadFile(walletFilepath+"pub.asc")
	check(err)
	return string(publicKeyPem)
}


func getSendAddress() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	sendAddress, err := reader.ReadString('\n')
	check(err)
	sendAddress = strip(sendAddress)

	if addressIsValid(sendAddress) == false {
		return "", errors.New("addresss not valid")
	}

	return sendAddress, nil
}


func getAmountToSend() float64 {
	reader := bufio.NewReader(os.Stdin)
	amountString, err := reader.ReadString('\n')
	amount, err := strconv.ParseFloat(strip(amountString), 64)
	check(err)

	return amount
}


func strip(raw_string string) string {
	return strings.TrimRight(raw_string, "\r\n")
}


func addressIsValid(addr string) bool {
	if len(addr) != 32 {
		return false
	}
	_, err := hex.DecodeString(addr)
	if err != nil {
		return false
	}
	return true
}


func constructTransactionPacket(toAddr string, fromAddr string, amount float64) coin.Transaction {
	var publicKey string
	if requestPublicKeyCacheExistance(fromAddr) {
		publicKey = ""
	} else {
		publicKey = loadPublicKeyPem()
	}

	type T = coin.Transaction
	t_packet := T{}

	t_packet.ToAddress = toAddr
	t_packet.FromAddress = fromAddr
	t_packet.Amount = amount
	t_packet.Timestamp = time.Now().String()
	t_packet.PublicKey = publicKey
	t_packet.Signature = signTransaction(t_packet, loadPrivateKeyPem())

	return t_packet
}


func signTransaction(tx coin.Transaction, privateKeyPem string) string {
	txString, _ := blockchain.Serialise(tx)
	privateKey, _ := pgp.ParsePrivateKeyFromPemStr(privateKeyPem)

	txSignature := pgp.SignMessage(txString, privateKey)
	txSignatureString := b64.StdEncoding.EncodeToString(txSignature)

	return txSignatureString
}


func broadcastTransactionToNetwork(tx coin.Transaction) {
	reqHeader := netpack.ConstructRequestHeader("wallet", "Transaction")
	transactionString, err := blockchain.Serialise(tx)
	check(err)
	packet := netpack.ConstructNetworkPacket(reqHeader, transactionString)
	packetString, _ := blockchain.Serialise(packet)

	for _, port := range nodeList {
		netpack.BroadcastPacket(packetString, port)
	}
}


func requestWalletBalance(walletAddr string) float64 {
	reqHeader := netpack.ConstructRequestHeader("wallet", "Balance")
	packet := netpack.ConstructNetworkPacket(reqHeader, walletAddr)
	packetString, _ := blockchain.Serialise(packet)
	balance := -1.0

	for _, port := range nodeList {
		success, response := netpack.BroadcastDuplexPacket(packetString, port)

		if success {
			balance, _ = strconv.ParseFloat(response.Body, 64)
			break
		}
	}
	
	return balance
}


func requestPublicKeyCacheExistance(walletAddress string) bool {
	reqHeader := netpack.ConstructRequestHeader("wallet", "PublicKeyInCache")
	packet := netpack.ConstructNetworkPacket(reqHeader, walletAddress)
	packetString, _ := blockchain.Serialise(packet)
	exists := false

	for _, port := range nodeList {
		success, response := netpack.BroadcastDuplexPacket(packetString, port)
		if success {
			if response.Body == "true" {
				exists = true
				break
			}
		}
	}
	
	return exists
}