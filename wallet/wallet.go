package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Sucks-To-Suck/LuncheonNetwork/blockchain"
	"github.com/Sucks-To-Suck/LuncheonNetwork/ellip"
	"github.com/Sucks-To-Suck/LuncheonNetwork/transactions"
	"github.com/Sucks-To-Suck/LuncheonNetwork/utilities"
	"golang.org/x/crypto/sha3"
)

type Wallet struct {
	chain   *blockchain.Blockchain
	mainKey ellip.MainKey
}

// Initialize a wallet by calling this function.
// Input is the blockchain the wallet is on.
// Returns a new wallet.
func Init(b *blockchain.Blockchain) Wallet {

	w := new(Wallet)

	w.chain = b

	return *w
}

// Scans the blockchain for the available balance of a publicKey.
// Returns the balance of the publicKey.
func (w *Wallet) ScanChainForBalance(pubKey string) (balance uint64) {

	// Scans the blockchain, starting from the newest block to the first
	for index := 0; index < len(w.chain.Blocks); index += 1 {

		// Check if they got the block reward (+10 makes the miner wait at least 10 blocks before it can be spent)
		if w.chain.Blocks[index].Miner == pubKey && (index+10) < int(w.chain.GetHeight()) {

			balance += w.chain.GetBlockReward(uint32(index))
		}

		// Check each tx in the block
		for txIndex := 0; txIndex < len(w.chain.Blocks[index].Txs); txIndex += 1 {

			if w.chain.Blocks[index].Txs[txIndex].TxTo == pubKey {

				balance += w.chain.Blocks[index].Txs[txIndex].Value
			}
		}
	}

	return balance
}

// Scans the blockchain for the available balance of a publicKey.
// Returns the balance of the publicKey.
func (w *Wallet) ScanChainForNonce(pubKey string) (nonce uint32) {

	// Scans the blockchain, starting from the newest block to the first
	for index := 0; index < len(w.chain.Blocks); index += 1 {

		// Check each tx in the block
		for txIndex := 0; txIndex < len(w.chain.Blocks[index].Txs); txIndex += 1 {

			if w.chain.Blocks[index].Txs[txIndex].TxFrom == pubKey {

				nonce += 1
			}
		}
	}

	return nonce
}

// This function creates a tx and verifys it.
// Inputs are the publicKey the tx is going to, and the amount of Luncheon that is being sent.
// Outputs are the tx, which if empty, means that the amount specified is not possible with your balance.
func (w *Wallet) CreateTx(toPub string, amount uint64) (tx transactions.LuTx) {

	// Say the tx is from you
	tx.TxFrom = w.mainKey.GetPubKeyStr()

	tx.TxTo = toPub
	tx.Value = amount

	tx.Nonce = w.ScanChainForNonce(tx.TxFrom)

	// Simple calculation to get a tx fee
	tx.Fee = uint64((tx.GetWeight() + 64) * 100) // The +64 is to add the weight of the signature

	txBytes, _ := json.Marshal(tx)

	_, sig := w.mainKey.SignMsg(txBytes)
	tx.Signature = hex.EncodeToString(sig)

	return tx
}

// Function calculates whether the tx input is valid or not.
// Input is the tx.
// Returns true if valid, false if not valid.
func (w *Wallet) VerifyTx(tx transactions.LuTx) bool {

	// If the tx has a spendable amount of coin from the persons balance
	if w.ScanChainForBalance(tx.TxFrom)-(tx.Value+tx.Fee) > 0 {

		return false
	}

	// If the tx has the wrong nonce value
	if tx.Nonce != w.ScanChainForNonce(tx.TxFrom) {

		return false
	}

	// Remove the sig from the tx and save it, as to get the tx hash input data
	signature, _ := hex.DecodeString(tx.Signature)
	tx.Signature = ""

	txBytes := tx.AsBytes()
	txHash := make([]byte, 32)
	pubKey, _ := hex.DecodeString(tx.TxFrom)

	sha3.ShakeSum256(txHash, txBytes)

	// If the signature is not valid
	// If this is true, than the tx is true
	return ellip.ValidateSig(pubKey, txHash, signature)
}

// Verifies of the block inputted is valid or not.
// Input is the block being verified. The second input is a bool that determines whether a block should have the same software version as you.
// Input true to have it check, false to have it just check the block normally.
// Returns true if it is valid, false if not valid.
func (w *Wallet) VerifyBlock(block *blockchain.Block, checkSoftwareVersion bool) bool {

	// If it is the genisis block
	if len(w.chain.Blocks) == 1 {

		return true
	}

	// Checks if the software version, if the func is told to do so
	if checkSoftwareVersion {

		if block.SoftwareVersion != utilities.SoftwareVersion {

			return false
		}
	}

	bytesUtil := new(utilities.ByteUtil)

	// Check the Block hash
	softwareVersion := []byte(block.SoftwareVersion)
	prevBlockHash, _ := hex.DecodeString(block.PrevHash)
	merkleRoot, _ := hex.DecodeString(block.MerkleRoot)
	blockTime := bytesUtil.Uint64toB(block.Timestamp)
	packedTargetBytes := bytesUtil.Uint32toB(block.PackedTarget)
	nonceBytes := bytesUtil.Uint32toB(block.Nonce)

	// Shove them together (into softwareVerion bc it is first declared)
	softwareVersion = append(softwareVersion, prevBlockHash...)
	softwareVersion = append(softwareVersion, merkleRoot...)
	softwareVersion = append(softwareVersion, packedTargetBytes...)
	softwareVersion = append(softwareVersion, blockTime...)
	softwareVersion = append(softwareVersion, nonceBytes...)

	hash := make([]byte, 32)

	// Hash the data
	sha3.ShakeSum256(hash, softwareVersion)

	// If the blockhash is invalid
	if hex.EncodeToString(hash) != block.BlockHash {

		fmt.Println(hex.EncodeToString(hash))
		return false
	}

	// Check if the block points to the previous block
	if block.PrevHash != w.chain.Blocks[w.chain.GetHeight()].BlockHash {

		return false
	}

	timeUtil := new(utilities.Time)

	// Check if the timestamp is valid
	// TODO: make more advanced
	if block.Timestamp < w.chain.Blocks[w.chain.GetHeight()].Timestamp || block.Timestamp > timeUtil.CurrentUnix() {

		return false
	}

	// Check if the target is correct
	if block.PackedTarget != w.chain.CalculatePackedTarget(uint(len(w.chain.Blocks))) {

		return false
	}

	// Check the merkle root
	if block.MerkleRoot != block.GetMerkleRoot() {

		return false
	}

	// Check the txs
	for index := 0; index < len(block.Txs); index += 1 {

		// If the tx is not valid, just remove it
		if !w.VerifyTx(block.Txs[index]) {

			block.RemoveTx(uint(index))
		}
	}

	return true
}

// Verifys whether the blockchain attached to the wallet is valid or not.
// Returns true if valid, false if invalid.
func (w *Wallet) VerifyBlockchain() bool {

	// Is only valid if no blocks are in the chain
	if len(w.chain.Blocks) == 0 {

		return true
	}

	//****
	// Check the genisis block:

	if len(w.chain.Blocks[0].Txs) != 0 {

		return false
	}

	if w.chain.Blocks[0].PackedTarget != 0x1d0fffff {

		return false
	}

	// Check the genisis block
	//****

	//****
	// Checks the rest of the blocks

	for blockIndex := 1; blockIndex < len(w.chain.Blocks); blockIndex += 1 {

		if !w.VerifyBlock(&w.chain.Blocks[blockIndex], false) {

			return false
		}
	}

	// Checks the rest of the blocks
	//****

	return true
}
