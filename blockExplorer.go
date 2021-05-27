// [-b] 	Can pretty print blocks given a block id (1, 45, 89, etc)
// [-s]		Stats
// can view the blockId of blocks that contain transactions
// can view the blockId of blocks that contain pgp public keys
// can view the number of coins in circulation
// can view the block height
// can view the blockIds of the blocks with the most transactions
// can view all wallet addresses in the blockchain
// can view how many coins have been moved in the entire blockchain
package main

import (
	"fmt"
	"flag"
	"strconv"
	"pocketcoin/blockchain"
	"pocketcoin/coin"
)


func main() {
	blockchainFolderPtr := flag.String("f", "", "Folder that stores the blockchain to explore")
	viewBlockPtr := flag.String("blk", "", "Block ID of a given block to view")
	blockHeightPtr := flag.Bool("h", false, "View the current block height")
	coinCirculationPtr := flag.Bool("c", false, "View number of coins in circulation")
	walletAddressListPtr := flag.Bool("w", false, "View all wallet addresses on the blockchain")
	minedBlocksCountPtr := flag.Bool("m", false, "View how many blocks each miner mined")
	publicKeyListPtr := flag.Bool("pub", false, "View all block IDs of blocks containing a PGP public key")
	transactionBlocksPtr := flag.Bool("t", false, "View all block IDs of blocks that contain transactions")
	walletBalancePtr := flag.Bool("b", false, "View the balance of all wallet addresses on the network")
	flag.Parse()

	blockchainFolder := *blockchainFolderPtr
	viewBlockId := *viewBlockPtr
	blockHeightFlag := *blockHeightPtr
	coinCirculationFlag := *coinCirculationPtr
	walletAddressListFlag := *walletAddressListPtr
	minedBlocksCountFlag := *minedBlocksCountPtr
	publicKeyListFlag := *publicKeyListPtr
	transactionBlocksFlag := *transactionBlocksPtr
	walletBalanceFlag := *walletBalancePtr

	if blockchainFolder == "" {
		fmt.Println("Missing command line argument [-f] - Folder that stores the blockchain to explore")
		return
	}

	blockchain.SetBlockchainFolder(blockchainFolder)

	if viewBlockId != "" {
		printBlock(viewBlockId)
	}
	if blockHeightFlag {
		printBlockHeight()
	}
	if coinCirculationFlag {
		printNumOfCoins()
	}
	if walletAddressListFlag {
		printAllWallets()
	}
	if minedBlocksCountFlag {
		printMinerStats()
	}
	if publicKeyListFlag {
		printBlocksWithPublicKey()
	}
	if transactionBlocksFlag {
		printBlocksWithTransactions()
	}
	if walletBalanceFlag {
		printAllWalletBalances()
	}
}


func printBlock(blockID string) {
	filename := "block_" + blockID + ".blk"
	blockString, err := blockchain.LoadBlock(filename)
	if err != nil {
		fmt.Println("Block specified not available!")
		return
	}

	fmt.Printf("\n%s\n", filename)
	block := blockchain.DeserialiseBlock(blockString)
	blockchain.PrettyPrint(block)
}


func printBlockHeight() {
	blockchainHeight := blockchain.Height()
	fmt.Printf("\nBlockchain Height: %d\n", blockchainHeight)
}


func printNumOfCoins() {
	height := blockchain.Height() + 1
	fmt.Printf("\nNumber of coins in circulation: %d\n", (height * 10))
}


func printAllWallets() {
	allWallets := getAllWalletAddresses()

	fmt.Println("\nAll wallet addresses found in the blockchain:")
	for _, address := range allWallets {
		fmt.Printf("    %s\n", address)
	}
}


func getAllWalletAddresses() []string {
	height := blockchain.Height()
	var allWallets []string

	for i:=0; i <= height; i++ {
		filename := "block_" + strconv.Itoa(i) + ".blk"
		blockString, _ := blockchain.LoadBlock(filename)
		block := blockchain.DeserialiseBlock(blockString)
		allWallets = append(allWallets, extractWalletAddresses(block)...)
	}

	return unique(allWallets)
}


func extractWalletAddresses(block coin.Block) []string {
	var addressList []string
	for _, tx := range block.Body {
		addressList = append(addressList, tx.ToAddress)
		if tx.FromAddress != "coinbase" {
			addressList = append(addressList, tx.FromAddress)
		}
	}

	return addressList
}


func unique(stringSlice []string) []string {
    keys := make(map[string]bool)
    list := []string{}	
    for _, entry := range stringSlice {
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }    
    return list
}


func printMinerStats() {
	fmt.Println("\nNumber of blocks each miner wallet address has mined")
	var minerMap = make(map[string]int)
	var minerExists = make(map[string]bool)

	height := blockchain.Height()

	for i:=0; i <= height; i++ {
		filename := "block_" + strconv.Itoa(i) + ".blk"
		blockString, _ := blockchain.LoadBlock(filename)
		block := blockchain.DeserialiseBlock(blockString)
		minerAddress := block.Body[0].ToAddress 
		if minerExists[minerAddress] {
			minerMap[minerAddress] += 1
		} else {
			minerExists[minerAddress] = true
			minerMap[minerAddress] = 1
		}
	}

	for key, value := range minerMap {
		fmt.Printf("  %s: %d\n", key, value)
	}

}



func printBlocksWithPublicKey() {
	height := blockchain.Height()
	fmt.Println("\nBlocks containing PGP public keys:")

	for i:=0; i <= height; i++ {
		filename := "block_" + strconv.Itoa(i) + ".blk"
		blockString, _ := blockchain.LoadBlock(filename)
		block := blockchain.DeserialiseBlock(blockString)
		if containsPublicKey(block) {
			fmt.Printf("    %s\n", filename)
		}
	}
}


func containsPublicKey(block coin.Block) bool {
	for _, tx := range block.Body {
		if tx.PublicKey != "" {
			return true
		}
	}
	return false
}


func printBlocksWithTransactions() {
	height := blockchain.Height()
	fmt.Println("\nBlocks containing transactions:")
	fmt.Println("  Block\t\t  Number of Transactions")
	fmt.Println("  --\t\t  --")

	for i:=0; i <= height; i++ {
		filename := "block_" + strconv.Itoa(i) + ".blk"
		blockString, _ := blockchain.LoadBlock(filename)
		block := blockchain.DeserialiseBlock(blockString)
		txBlock, txCount := containsTransactions(block)
		if txBlock {
			fmt.Printf("  %s\t  %d\n", filename, txCount)
		}
	}
}


func containsTransactions(block coin.Block) (bool, int) {
	if len(block.Body) > 1 {
		return true, len(block.Body)-1
	}
	return false, 0
}


func printAllWalletBalances() {
	fmt.Println("\nAll wallet balances:")
	allWallets := getAllWalletAddresses()

	for _, walletAddress := range allWallets {
		fmt.Printf("  %s: %f\n", walletAddress, getWalletBalance(walletAddress))
	}

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