package consensus

// This is to implement Expected Consensus protocol
// See: https://github.com/filecoin-project/specs/blob/master/expected-consensus.md

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	"github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/metrics/tracing"
	"github.com/filecoin-project/go-filecoin/proofs"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
)

var (
	// ErrStateRootMismatch is returned when the computed state root doesn't match the expected result.
	ErrStateRootMismatch = errors.New("blocks state root does not match computed result")
	// ErrInvalidBase is returned when the chain doesn't connect back to a known good block.
	ErrInvalidBase = errors.New("block does not connect to a known good chain")
	// ErrUnorderedTipSets is returned when weight and minticket are the same between two tipsets.
	ErrUnorderedTipSets = errors.New("trying to order two identical tipsets")
)

// Expected implements expected consensus.
type Expected struct {
	// PwrTableView provides miner and total power for the EC chain weight
	// computation.
	PwrTableView PowerTableView

	// validator provides a set of methods used to validate a block.
	validator BlockValidator

	// cstore is used for loading state trees during message running.
	cstore *hamt.CborIpldStore

	// bstore contains data referenced by actors within the state
	// during message running.  Additionally bstore is used for
	// accessing the power table.
	bstore blockstore.Blockstore

	// processor is what we use to process messages and pay rewards
	processor Processor

	genesisCid cid.Cid

	verifier proofs.Verifier

	logger logging.EventLogger
}

// Ensure Expected satisfies the Protocol interface at compile time.
var _ Protocol = (*Expected)(nil)

// NewExpected is the constructor for the Expected consenus.Protocol module.
func NewExpected(cs *hamt.CborIpldStore, bs blockstore.Blockstore, processor Processor, v BlockValidator, pt PowerTableView, gCid cid.Cid, verifier proofs.Verifier) *Expected {
	return &Expected{
		PwrTableView: pt,

		bstore:     bs,
		cstore:     cs,
		genesisCid: gCid,
		logger:     logging.Logger("consensus/expected"),
		processor:  processor,
		validator:  v,
		verifier:   verifier,
	}
}

// Weight returns the EC weight of this TipSet in uint64 encoded fixed point
// representation.
func (c *Expected) Weight(ctx context.Context, ts types.TipSet, pSt state.Tree) (uint64, error) {
	if ts.Len() == 1 && ts.At(0).Cid().Equals(c.genesisCid) {
		return uint64(0), nil
	}
	// Compute parent weight.
	parentW, err := ts.ParentWeight()
	if err != nil {
		return uint64(0), err
	}

	w, err := types.FixedToBig(parentW)
	if err != nil {
		return uint64(0), err
	}
	// Each block in the tipset adds ECV + ECPrm * miner_power to parent weight.
	totalBytes, err := c.PwrTableView.Total(ctx, pSt, c.bstore)
	if err != nil {
		return uint64(0), err
	}
	floatTotalBytes := new(big.Float).SetInt(totalBytes.BigInt())
	floatECV := new(big.Float).SetInt64(int64(ECV))
	floatECPrM := new(big.Float).SetInt64(int64(ECPrM))
	for _, blk := range ts.ToSlice() {
		minerBytes, err := c.PwrTableView.Miner(ctx, pSt, c.bstore, blk.Miner)
		if err != nil {
			return uint64(0), err
		}
		floatOwnBytes := new(big.Float).SetInt(minerBytes.BigInt())
		wBlk := new(big.Float)
		wBlk.Quo(floatOwnBytes, floatTotalBytes)
		wBlk.Mul(wBlk, floatECPrM) // Power addition
		wBlk.Add(wBlk, floatECV)   // Constant addition
		w.Add(w, wBlk)
	}
	return types.BigToFixed(w)
}

// IsHeavier returns true if tipset a is heavier than tipset b, and false
// vice versa.  In the rare case where two tipsets have the same weight ties
// are broken by taking the tipset with the smallest ticket.  In the event that
// tickets are the same, IsHeavier will break ties by comparing the
// concatenation of block cids in the tipset.
// TODO BLOCK CID CONCAT TIE BREAKER IS NOT IN THE SPEC AND SHOULD BE
// EVALUATED BEFORE GETTING TO PRODUCTION.
func (c *Expected) IsHeavier(ctx context.Context, a, b types.TipSet, aSt, bSt state.Tree) (bool, error) {
	aW, err := c.Weight(ctx, a, aSt)
	if err != nil {
		return false, err
	}
	bW, err := c.Weight(ctx, b, bSt)
	if err != nil {
		return false, err
	}

	// Without ties pass along the comparison.
	if aW != bW {
		return aW > bW, nil
	}

	// To break ties compare the min tickets.
	aTicket, err := a.MinTicket()
	if err != nil {
		return false, err
	}
	bTicket, err := b.MinTicket()
	if err != nil {
		return false, err
	}

	cmp := bytes.Compare(bTicket, aTicket)
	if cmp != 0 {
		// a is heavier if b's ticket is greater than a's ticket.
		return cmp == 1, nil
	}

	// Tie break on cid ids.
	// TODO: I think this is drastically impacted by number of blocks in tipset
	// i.e. bigger tipset is always heavier.  Not sure if this is ok, need to revist.
	cmp = strings.Compare(a.String(), b.String())
	if cmp == 0 {
		// Caller is mistakenly calling on two identical tipsets.
		return false, ErrUnorderedTipSets
	}
	return cmp == 1, nil
}

// RunStateTransition is the chain transition function that goes from a
// starting state and a tipset to a new state.  It errors if the tipset was not
// mined according to the EC rules, or if running the messages in the tipset
// results in an error.
func (c *Expected) RunStateTransition(ctx context.Context, ts types.TipSet, ancestors []types.TipSet, pSt state.Tree) (st state.Tree, err error) {
	ctx, span := trace.StartSpan(ctx, "Expected.RunStateTransition")
	span.AddAttributes(trace.StringAttribute("tipset", ts.String()))
	defer tracing.AddErrorEndSpan(ctx, span, &err)

	for i := 0; i < ts.Len(); i++ {
		if err := c.ValidateSemantic(ctx, ancestors[0].At(0), ts.At(i)); err != nil {
			return nil, err
		}
	}

	if err := c.validateMining(ctx, pSt, ts, ancestors[0]); err != nil {
		return nil, err
	}

	vms := vm.NewStorageMap(c.bstore)
	st, err = c.runMessages(ctx, pSt, vms, ts, ancestors)
	if err != nil {
		return nil, err
	}
	err = vms.Flush()
	if err != nil {
		return nil, err
	}
	return st, nil
}

// ValidateSyntax validates a single block is correctly formed.
func (c *Expected) ValidateSyntax(ctx context.Context, b *types.Block) error {
	return c.validator.ValidateSyntax(ctx, b)
}

// ValidateSemantic validates a block is correctly derived from its parent.
func (c *Expected) ValidateSemantic(ctx context.Context, child, parent *types.Block) error {
	return c.validator.ValidateSemantic(ctx, child, parent)
}

// validateMining checks validity of the block ticket, proof, and miner address.
//    Returns an error if:
//    	* any tipset's block was mined by an invalid miner address.
//      * the block proof is invalid for the challenge
//      * the block ticket fails the power check, i.e. is not a winning ticket
//    Returns nil if all the above checks pass.
// See https://github.com/filecoin-project/specs/blob/master/mining.md#chain-validation
func (c *Expected) validateMining(ctx context.Context, st state.Tree, ts types.TipSet, parentTs types.TipSet) error {
	for i := 0; i < ts.Len(); i++ {
		blk := ts.At(i)
		// TODO: Also need to validate BlockSig

		// TODO: Once we've picked a delay function (see #2119), we need to
		// verify its proof here. The proof will likely be written to a field on
		// the mined block.

		// See https://github.com/filecoin-project/specs/blob/master/mining.md#ticket-checking
		result, err := IsWinningTicket(ctx, c.bstore, c.PwrTableView, st, blk.Ticket, blk.Miner)
		if err != nil {
			return errors.Wrap(err, "can't check for winning ticket")
		}

		if !result {
			return errors.New("not a winning ticket")
		}
	}
	return nil
}

// runMessages applies the messages of all blocks within the input
// tipset to the input base state.  Messages are applied block by
// block with blocks sorted by their ticket bytes.  The output state must be
// flushed after calling to guarantee that the state transitions propagate.
//
// An error is returned if individual blocks contain messages that do not
// lead to successful state transitions.  An error is also returned if the node
// faults while running aggregate state computation.
func (c *Expected) runMessages(ctx context.Context, st state.Tree, vms vm.StorageMap, ts types.TipSet, ancestors []types.TipSet) (state.Tree, error) {
	var cpySt state.Tree

	// TODO: don't process messages twice
	for i := 0; i < ts.Len(); i++ {
		blk := ts.At(i)
		cpyCid, err := st.Flush(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		// state copied so changes don't propagate between block validations
		cpySt, err = state.LoadStateTree(ctx, c.cstore, cpyCid, builtin.Actors)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}

		receipts, err := c.processor.ProcessBlock(ctx, cpySt, vms, blk, ancestors)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		// TODO: check that receipts actually match
		if len(receipts) != len(blk.MessageReceipts) {
			return nil, fmt.Errorf("found invalid message receipts: %v %v", receipts, blk.MessageReceipts)
		}

		outCid, err := cpySt.Flush(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "error validating block state")
		}
		if !outCid.Equals(blk.StateRoot) {
			return nil, ErrStateRootMismatch
		}
	}
	if ts.Len() <= 1 { // block validation state == aggregate parent state
		return cpySt, nil
	}
	// multiblock tipsets require reapplying messages to get aggregate state
	// NOTE: It is possible to optimize further by applying block validation
	// in sorted order to reuse first block transitions as the starting state
	// for the tipSetProcessor.
	_, err := c.processor.ProcessTipSet(ctx, st, vms, ts, ancestors)
	if err != nil {
		return nil, errors.Wrap(err, "error validating tipset")
	}
	return st, nil
}
