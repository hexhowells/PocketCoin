<p align="center">
  <img src="https://github.com/hexhowells/PocketCoin/blob/main/logo.png" width=80%>
</p>
PocketCoin is a small and simple cryptocurrency / blockchain designed to learn about the technology behind blockchains as well as to learn the Go programming language and isn't a full implimentation that can or should be used.

Overview
-----

The blockchain network is comprised of 3 node types:
- ```wallet.go``` - used to broadcast transactions and get wallet balalnces
- ```node.go``` - used to broadcast transactions and mined blocks throught the network
- ```miner.go```  - used to mine new blocks

Includes the ```shards/``` folder, this contains copies of the blockchain so that a whole network can be run on the same machine without sharing a blockchain, This code is designed to be run on a single machine as node ip's are simulated using different ports.

```packge/pocketcoin/``` contains a few modules that are used in the network, these should be placed in the your ```/go/src``` folder.

Features / Notes
-----
- Miner nodes can sync their blockchain with nodes if missing any blocks
- Wallet addresses are truncated SHA256 hashes of the wallets pgp public key
- Smallest unit of PocketCoin is 0.000001ρ
- Blocks are limited to 10 transactions (not including the coinbase transaction)
- Uses the Account Balance Model over UTXO
- Miners are rewarded a fixed amount of 10 coins for mining a block
- Mining a block takes anywhere between ~2s to ~6m (little too volatile but it'll suffice)
- Transactions are pgp signed for verification
- Pgp public keys are only broadcasted on the wallet's first transaction (pointers to a wallets public key in the blockchain are cached to reduce the blockchain size)
- Can view individual blocks, balances, and stats about the blockchain using ```blockExplorer.go```
- The mining code is not fast and could be greatly optimised thus requiring more difficult targets, current implimentation works fine for learning purposes though
- MerkleRoot in the block header is just the hash of the transactions in the body

Usage
-----
Start ```LaunchNetwork.bat``` to launch the network locally, batch file launches a few nodes and miners.
#### Node.go
```
    -f                      Specify the folder containing the blockchain.
    -p                      Specify the port that the node runs on.
```

#### Miner.go
```
    -f                      Specify the folder containing the blockchain.
    -p                      Specify the port that the miner node runs on.
    -w                      The wallet address to send the mined rewards to.
```

#### Wallet.go
```
    -f                      Specify the folder containing the wallet (wallet address, pgp keys)
    -b                      Display the wallets balance (needs to connect to a node)
    -t                      Create and send a transaction
    -w                      Display the wallet's address
    -n                      Create a new wallet address, deletes previously stored wallet address
```

#### blockExplorer.go
```
    -f                      Folder that stores the blockchain to be explored
    -b                      View the balance of all the wallets found on the network
    -blk                    Block ID of a given block to view
    -c                      View the number of coins currently in circulation
    -h                      View the current block height
    -m                      View how many blocks each miner wallet has mined
    -pub                    View all block IDs of blocks containing PGP public keys
    -t                      View all block IDs of blocks containing transactions
    -w                      View all wallet addresses found on the blockchain
```

Example Block
-----
#### block_99.blk
    {
            "Hash": "0000002b99d8c5af47a925efb4f56b1d4b5843fd6c07a13a3a098702835924b0",
            "Header": {
                    "Version": 0.1,
                    "BlockId": "99",
                    "PrevBlockHash": "00000010f7fc8ccfdfb0319a806a9679279fe66e5006b3d0f86632a2d63ee893",
                    "MerkleRoot": "b45f4b7dde4992e9c717471e2f67be96390ea536aa3918547c632434c951f72f",
                    "Timestamp": "2021-05-15 17:24:10.663253 +0100 BST m=+561.423202701",
                    "Nonce": 1650861,
                    "TargetBits": 3.450873173395282e+69
            },
            "Body": [
                    {
                            "Amount": 10,
                            "ToAddress": "98f7482b7b93244e5a3e30e9bad76107",
                            "FromAddress": "coinbase",
                            "Signature": "",
                            "PublicKey": "",
                            "Timestamp": "2021-05-15 17:24:10.663253 +0100 BST m=+561.423202701"
                    },
                    {
                            "Amount": 0.000001,
                            "ToAddress": "da7dad6966a0692960480de5b3d0b7bd",
                            "FromAddress": "e50f8a0089db3ce621c492325474b8e6",
                            "Signature": "ViBw ... 98MA=",
                            "PublicKey": "",
                            "Timestamp": "2021-05-15 17:20:46.2113253 +0100 BST m=+5.265946101"
                    }
            ]
    }
Future Additions / Improvements
-----
- Impliment a Merkle Tree
- Look into implimenting UTXO (Unspent Transaction Output)
- More robust handling of blockchain forking and orphan blocks
- More stable block mining times
