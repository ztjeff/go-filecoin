package validation

import (
	"context"

	"github.com/filecoin-project/chain-validation/pkg/chain"
	"github.com/filecoin-project/chain-validation/pkg/state"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
)

// Applier applies messages to state trees and storage.
type Applier struct {
	processor *consensus.DefaultProcessor
}

var _ chain.Applier = &Applier{}

func NewApplier() *Applier {
	return &Applier{consensus.NewDefaultProcessor()}
}

func (a *Applier) ApplyMessage(tree state.Tree, storage state.StorageMap, eCtx *chain.ExecutionContext,
	message interface{}) (state.Tree, chain.MessageReceipt, error) {
	ctx := context.TODO()
	stateTree := tree.(*stateTreeWrapper).Tree
	vms := storage.(*storageMapWrapper).StorageMap
	msg := message.(*types.SignedMessage)
	minerOwner, err := address.NewFromBytes([]byte(eCtx.MinerOwner))
	if err != nil {
		return nil, chain.MessageReceipt{}, err
	}
	blockHeight := types.NewBlockHeight(eCtx.Epoch)
	gasTracker := vm.NewGasTracker()
	// Providing direct access to blockchain structures is very difficult and expected to go away.
	var ancestors []types.TipSet

	amr, err := a.processor.ApplyMessage(ctx, stateTree, vms, msg, minerOwner, blockHeight, gasTracker, ancestors)
	if err != nil {
		return nil, chain.MessageReceipt{}, err
	}
	// Go-filecoin has some messed-up nested array return value.
	retVal := []byte{}
	if len(amr.Receipt.Return) > 0 {
		retVal = amr.Receipt.Return[0]
	}
	mr := chain.MessageReceipt{
		ExitCode:    amr.Receipt.ExitCode,
		ReturnValue: retVal,
		// Go-filecoin returns the gas cost rather than gas unit consumption :-(
		GasUsed:     state.GasUnit(amr.Receipt.GasAttoFIL.AsBigInt().Uint64()),
	}

	// The intention of this method is to leave the input tree untouched and return a new one.
	// FIXME the state.Tree implements a mutable tree - it's impossible to hold on to the prior state
	// We might need to implement our own state tree to achieve that, or make extensive improvement to go-filecoin.
	return tree, mr, nil
}
