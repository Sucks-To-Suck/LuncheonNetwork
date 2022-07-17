package transactions

import (
	"errors"

	"github.com/Sucks-To-Suck/LuncheonNetwork/utilities"
)

type Weight struct {
	weight uint32

	util utilities.Util
}

func (w *Weight) WeightPLuX(tx *PLuX) error {

	// Adds the byte count for the two uint64's
	w.weight += (8 * 2)

	// Adds the byte count for storing this weight
	w.weight += 4

	// If no winning miner is stored in the transaction
	if tx.luckyMiner == nil {

		return errors.New("Cannot create a transaction with no id for the winning miner!")
	}

	// Adds the bytes of the miner who discovers the block
	w.weight += uint32(len(tx.luckyMiner))

	return nil
}