package cst

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/chain"
	"github.com/filecoin-project/go-filecoin/sampling"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
)

type chainReadWriter interface {
	GetHead() types.TipSetKey
	GetTipSet(types.TipSetKey) (types.TipSet, error)
	GetTipSetState(context.Context, types.TipSetKey) (state.Tree, error)
	SetHead(context.Context, types.TipSet) error
}

// ChainStateReadWriter composes a:
// ChainReader providing read access to the chain and its associated state.
// ChainWriter providing write access to the chain head.
type ChainStateReadWriter struct {
	readWriter      chainReadWriter
	cst             *hamt.CborIpldStore // Provides chain blocks and state trees.
	messageProvider chain.MessageProvider
}

// NewChainStateReadWriter returns a new ChainStateReadWriter.
func NewChainStateReadWriter(crw chainReadWriter, messages chain.MessageProvider, cst *hamt.CborIpldStore) *ChainStateReadWriter {
	return &ChainStateReadWriter{
		readWriter:      crw,
		cst:             cst,
		messageProvider: messages,
	}
}

// Head returns the head tipset
func (chn *ChainStateReadWriter) Head() types.TipSetKey {
	return chn.readWriter.GetHead()
}

// GetTipSet returns the tipset at the given key
func (chn *ChainStateReadWriter) GetTipSet(key types.TipSetKey) (types.TipSet, error) {
	return chn.readWriter.GetTipSet(key)
}

// Ls returns an iterator over tipsets from head to genesis.
func (chn *ChainStateReadWriter) Ls(ctx context.Context) (*chain.TipsetIterator, error) {
	ts, err := chn.readWriter.GetTipSet(chn.readWriter.GetHead())
	if err != nil {
		return nil, err
	}
	return chain.IterAncestors(ctx, chn.readWriter, ts), nil
}

// GetBlock gets a block by CID
func (chn *ChainStateReadWriter) GetBlock(ctx context.Context, id cid.Cid) (*types.Block, error) {
	var out types.Block
	err := chn.cst.Get(ctx, id, &out)
	return &out, err
}

// GetMessages gets a message collection by CID.
func (chn *ChainStateReadWriter) GetMessages(ctx context.Context, id cid.Cid) ([]*types.SignedMessage, error) {
	return chn.messageProvider.LoadMessages(ctx, id)
}

// GetReceipts gets a receipt collection by CID.
func (chn *ChainStateReadWriter) GetReceipts(ctx context.Context, id cid.Cid) ([]*types.MessageReceipt, error) {
	return chn.messageProvider.LoadReceipts(ctx, id)
}

// SampleRandomness samples randomness from the chain at the given height.
func (chn *ChainStateReadWriter) SampleRandomness(ctx context.Context, sampleHeight *types.BlockHeight) ([]byte, error) {
	head := chn.readWriter.GetHead()
	headTipSet, err := chn.readWriter.GetTipSet(head)
	if err != nil {
		return nil, err
	}
	tipSetBuffer, err := chain.GetRecentAncestors(ctx, headTipSet, chn.readWriter, sampleHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get recent ancestors")
	}

	return sampling.SampleChainRandomness(sampleHeight, tipSetBuffer)
}

// GetActor returns an actor from the latest state on the chain
func (chn *ChainStateReadWriter) GetActor(ctx context.Context, addr address.Address) (*actor.Actor, error) {
	return chn.GetActorAt(ctx, chn.readWriter.GetHead(), addr)
}

// GetActorAt returns an actor at a specified tipset key.
func (chn *ChainStateReadWriter) GetActorAt(ctx context.Context, tipKey types.TipSetKey, addr address.Address) (*actor.Actor, error) {
	st, err := chn.readWriter.GetTipSetState(ctx, tipKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load latest state")
	}

	actr, err := st.GetActor(ctx, addr)
	if err != nil {
		return nil, errors.Wrapf(err, "no actor at address %s", addr)
	}
	return actr, nil
}

// LsActors returns a channel with actors from the latest state on the chain
func (chn *ChainStateReadWriter) LsActors(ctx context.Context) (<-chan state.GetAllActorsResult, error) {
	st, err := chn.readWriter.GetTipSetState(ctx, chn.readWriter.GetHead())
	if err != nil {
		return nil, err
	}
	return state.GetAllActors(ctx, st), nil
}

// SetHead sets `key` as the new head of this chain iff it exists in the nodes chain store.
func (chn *ChainStateReadWriter) SetHead(ctx context.Context, key types.TipSetKey) error {
	headTs, err := chn.readWriter.GetTipSet(key)
	if err != nil {
		return err
	}
	return chn.readWriter.SetHead(ctx, headTs)
}
