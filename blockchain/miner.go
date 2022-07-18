package blockchain

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/Sucks-To-Suck/LuncheonNetwork/utilities"
	"golang.org/x/crypto/sha3"
)

// The struct that handles the mining. Uses the shake256 varient of sha3 for hashing.
type Miner struct {
	inputBlockBytes []byte
	packedTarget    uint32

	hashData       []byte
	currentHash    []byte
	unpackedTarget []byte

	util     utilities.ByteUtil
	unpacker utilities.TargetUnpacker
	utilTime utilities.Time
}

// This function tells the miner what target to mine to. Returns an error if once occurs.
func (m *Miner) inputTarget(inputTarget uint32) error {

	// 0 is an invalid target, and this handles that
	if inputTarget == 0 {

		return errors.New("cannot input target 0")
	}

	// Is the target higher than the max allowed target?
	if inputTarget > 0x1dffffff { // TODO: Find a better value and define as const elsewhere

		return errors.New("target is to large") // TODO: Have it print max target
	}

	m.packedTarget = inputTarget

	return nil
}

// Starts the miner. Will return a byte array of the valid hash once discovered. Also returns an error if once occured.
func (m *Miner) Start(b Block) (Block, error) {

	// Get the block as bytes for mining
	m.inputBlockBytes = b.ParseBlockToBytes()

	// No block data?
	if m.inputBlockBytes == nil {

		return b, errors.New("please input a block with data inside it")
	}

	// Unpack the target stored in the block
	unpackErr := m.inputTarget(b.PackedTarget)

	// If an error occured
	if unpackErr != nil {

		panic(unpackErr)
	}

	// Gets the unpacked target with the unpacker struct
	m.unpackedTarget = m.unpacker.UnpackAsBytes(m.packedTarget)

	fmt.Println("Mining Starting!")

	// The actual mining process
	for b.Nonce = 0; b.Nonce <= 0xFFFFFFFF; b.Nonce++ {

		// Set the timestamp in the block
		b.SetTimestamp(uint64(m.utilTime.CurrentUnix()))

		// Create the input bytes for the hash, and add the nonce
		m.hashData = append(m.inputBlockBytes, m.util.Uint32toB(b.Nonce)...)

		// Init the size of the hash
		m.currentHash = make([]byte, 32)

		// Hash the data
		sha3.ShakeSum256(m.currentHash, m.hashData)

		// Was the solution found?
		if bytes.Compare(m.currentHash, m.unpackedTarget) != 1 {

			// Set the block hash to the winning hash
			b.SetBlockHash(m.currentHash)

			return b, nil
		}

		// Prints stats every 10 MH
		if b.Nonce%10000000 == 0 {

			fmt.Println("Mining...")
			fmt.Printf("Target: %x\n", m.unpackedTarget)
			fmt.Printf("Last Hash: %x\n", m.currentHash)
		}
	}

	return b, errors.New("you have reached the end of the defined search space! Impressive")
}
