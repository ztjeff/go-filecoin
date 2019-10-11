package porcelain

import (
	"context"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/exec"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-filecoin/types"
)

type chainHeadPlumbing interface {
	ChainHeadKey() types.TipSetKey
	ChainTipSet(key types.TipSetKey) (types.TipSet, error)
}

// ChainHead gets the current head tipset from plumbing.
func ChainHead(plumbing chainHeadPlumbing) (types.TipSet, error) {
	return plumbing.ChainTipSet(plumbing.ChainHeadKey())
}

type fullBlockPlumbing interface {
	ChainGetBlock(context.Context, cid.Cid) (*types.Block, error)
	ChainGetMessages(context.Context, cid.Cid) ([]*types.SignedMessage, error)
	ChainGetReceipts(context.Context, cid.Cid) ([]*types.MessageReceipt, error)
}

// GetFullBlock returns a full block: header, messages, receipts.
func GetFullBlock(ctx context.Context, plumbing fullBlockPlumbing, id cid.Cid) (*types.FullBlock, error) {
	var out types.FullBlock
	var err error

	out.Header, err = plumbing.ChainGetBlock(ctx, id)
	if err != nil {
		return nil, err
	}

	out.Messages, err = plumbing.ChainGetMessages(ctx, out.Header.Messages)
	if err != nil {
		return nil, err
	}

	out.Receipts, err = plumbing.ChainGetReceipts(ctx, out.Header.MessageReceipts)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

type messagePlumbing interface {
	ChainStateRoot(baseKey types.TipSetKey) (cid.Cid, error)
	ChainTipSet(key types.TipSetKey) (types.TipSet, error)
	VMState(ctx context.Context, stateRoot cid.Cid, bh *types.BlockHeight) (consensus.VMState, error)
}

// MessageQuery pulls actor state for a baseKey and interrogates it
func MessageQuery(ctx context.Context, baseKey types.TipSetKey, api messagePlumbing, optFrom, to address.Address, method string, params ...interface{}) ([][]byte, error) {
	vmState, err := VMState(ctx, api, baseKey)
	if err != nil {
		return [][]byte{}, err
	}
	return vmState.Query(ctx, optFrom, to, method, params...)
}

// GetActorSignature gets the signature of an actor at the correct protocol version for a tipset
func GetActorSignature(ctx context.Context, api messagePlumbing, baseKey types.TipSetKey, actorAddr address.Address, method string) (_ *exec.FunctionSignature, err error) {
	vmState, err := VMState(ctx, api, baseKey)
	if err != nil {
		return nil, err
	}
	return vmState.GetActorSignature(ctx, actorAddr, method)
}

func VMState(ctx context.Context, api messagePlumbing, baseKey types.TipSetKey) (consensus.VMState, error) {
	// get state root and block height for tipset
	stateRoot, err := api.ChainStateRoot(baseKey)
	if err != nil {
		return consensus.VMState{}, err
	}

	ts, err := api.ChainTipSet(baseKey)
	if err != nil {
		return consensus.VMState{}, err
	}
	bh, err := ts.Height()
	if err != nil {
		return consensus.VMState{}, err
	}
	return api.VMState(ctx, stateRoot, types.NewBlockHeight(bh))
}
