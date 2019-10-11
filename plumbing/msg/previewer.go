package msg

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/types"
)

// Abstracts over a store of blockchain state.
type previewerChainReader interface {
	GetHead() types.TipSetKey
	GetTipSetStateRoot(types.TipSetKey) (cid.Cid, error)
	GetTipSet(types.TipSetKey) (types.TipSet, error)
}

type messagePreviewer interface {
	// PreviewQueryMethod estimates the amount of gas that will be used by a method
	PreviewQueryMethod(ctx context.Context, vmState consensus.VMState, to address.Address, method string, params []byte, from address.Address) (types.GasUnits, error)
}

// Previewer calculates the amount of Gas needed for a command
type Previewer struct {
	// To get the head tipset state root.
	chainReader previewerChainReader
	// To accsss vm state
	vmStateStore consensus.VMStateStore
	// To to preview messages
	processor messagePreviewer
}

// NewPreviewer constructs a Previewer.
func NewPreviewer(chainReader previewerChainReader, vmStateStore consensus.VMStateStore, processor messagePreviewer) *Previewer {
	return &Previewer{chainReader, vmStateStore, processor}
}

// Preview sends a read-only message to an actor.
func (p *Previewer) Preview(ctx context.Context, optFrom, to address.Address, method string, params ...interface{}) (types.GasUnits, error) {
	encodedParams, err := abi.ToEncodedValues(params...)
	if err != nil {
		return types.NewGasUnits(0), errors.Wrap(err, "failed to encode message params")
	}

	headKey := p.chainReader.GetHead()
	head, err := p.chainReader.GetTipSet(headKey)
	if err != nil {
		return types.NewGasUnits(0), errors.Wrap(err, "failed to get head tipset ")
	}
	blockHeight, err := head.Height()
	if err != nil {
		return types.NewGasUnits(0), errors.Wrap(err, "failed to get head tipset height")
	}
	stateRoot, err := p.chainReader.GetTipSetStateRoot(headKey)
	if err != nil {
		return types.NewGasUnits(0), errors.Wrap(err, "could not get tipset for head")
	}
	vmState, err := p.vmStateStore.State(ctx, stateRoot, types.NewBlockHeight(blockHeight))
	if err != nil {
		return types.NewGasUnits(0), errors.Wrap(err, "could not get vm state for head")
	}

	usedGas, err := p.processor.PreviewQueryMethod(ctx, vmState, to, method, encodedParams, optFrom)
	if err != nil {
		return types.NewGasUnits(0), errors.Wrap(err, "query method returned an error")
	}
	return usedGas, nil
}
