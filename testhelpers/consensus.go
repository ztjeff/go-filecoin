package testhelpers

import (
	"context"
	"testing"

	cid "github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/types"
)

// RequireNewTipSet instantiates and returns a new tipset of the given blocks
// and requires that the setup validation succeed.
func RequireNewTipSet(t *testing.T, blks ...*types.Block) types.TipSet {
	ts, err := types.NewTipSet(blks...)
	require.NoError(t, err)
	return ts
}

// NewValidTestBlockFromTipSet creates a block for when proofs & power table don't need
// to be correct
func NewValidTestBlockFromTipSet(baseTipSet types.TipSet, stateRootCid cid.Cid, height uint64, minerAddr address.Address, minerWorker address.Address, signer types.Signer) (*types.Block, error) {
	electionProof := consensus.MakeFakeElectionProofForTest()
	ticket := consensus.MakeFakeTicketForTest()

	b := &types.Block{
		Miner:         minerAddr,
		Tickets:       []types.Ticket{ticket},
		Parents:       baseTipSet.Key(),
		ParentWeight:  types.Uint64(10000 * height),
		Height:        types.Uint64(height),
		StateRoot:     stateRootCid,
		ElectionProof: electionProof,
	}
	sig, err := signer.SignBytes(b.SignatureData(), minerWorker)
	if err != nil {
		return nil, err
	}
	b.BlockSig = sig

	return b, nil
}

// MakeRandomPoStProofForTest creates a random proof.
func MakeRandomPoStProofForTest() types.PoStProof {
	proofSize := types.OnePoStProofPartition.ProofLen()
	p := MakeRandomBytes(proofSize)
	p[0] = 42
	poStProof := make([]byte, proofSize)
	for idx, elem := range p {
		poStProof[idx] = elem
	}
	return poStProof
}

// FakeSignedMessageValidator is a validator that doesn't validate to simplify message creation in tests.
type FakeSignedMessageValidator struct{}

var _ consensus.SignedMessageValidator = (*FakeSignedMessageValidator)(nil)

// Validate always returns nil
func (tsmv *FakeSignedMessageValidator) Validate(ctx context.Context, msg *types.SignedMessage, fromActor *actor.Actor) error {
	return nil
}

// FakeBlockRewarder is a rewarder that doesn't actually add any rewards to simplify state tracking in tests
type FakeBlockRewarder struct{}

var _ consensus.BlockRewarder = (*FakeBlockRewarder)(nil)

// BlockReward is a noop
func (tbr *FakeBlockRewarder) BlockReward(ctx context.Context, vmState consensus.VMState, minerAddr address.Address) error {
	// do nothing to keep state root the same
	return nil
}

// GasReward does nothing
func (tbr *FakeBlockRewarder) GasReward(ctx context.Context, vmState consensus.VMState, minerAddr address.Address, msg *types.SignedMessage, cost types.AttoFIL) error {
	// do nothing to keep state root the same
	return nil
}

// FakeBlockValidator passes everything as valid
type FakeBlockValidator struct{}

// NewFakeBlockValidator createas a FakeBlockValidator that passes everything as valid.
func NewFakeBlockValidator() *FakeBlockValidator {
	return &FakeBlockValidator{}
}

// ValidateSemantic does nothing.
func (fbv *FakeBlockValidator) ValidateSemantic(ctx context.Context, child *types.Block, parents *types.TipSet) error {
	return nil
}

// ValidateSyntax does nothing.
func (fbv *FakeBlockValidator) ValidateSyntax(ctx context.Context, blk *types.Block) error {
	return nil
}

// ValidateMessagesSyntax does nothing
func (fbv *FakeBlockValidator) ValidateMessagesSyntax(ctx context.Context, messages []*types.SignedMessage) error {
	return nil
}

// ValidateReceiptsSyntax does nothing
func (fbv *FakeBlockValidator) ValidateReceiptsSyntax(ctx context.Context, receipts []*types.MessageReceipt) error {
	return nil
}

// StubBlockValidator is a mockable block validator.
type StubBlockValidator struct {
	syntaxStubs   map[cid.Cid]error
	semanticStubs map[cid.Cid]error
}

// NewStubBlockValidator creates a StubBlockValidator that allows errors to configured
// for blocks passed to the Validate* methods.
func NewStubBlockValidator() *StubBlockValidator {
	return &StubBlockValidator{
		syntaxStubs:   make(map[cid.Cid]error),
		semanticStubs: make(map[cid.Cid]error),
	}
}

// ValidateSemantic returns nil or error for stubbed block `child`.
func (mbv *StubBlockValidator) ValidateSemantic(ctx context.Context, child *types.Block, parents *types.TipSet) error {
	return mbv.semanticStubs[child.Cid()]
}

// ValidateSyntax return nil or error for stubbed block `blk`.
func (mbv *StubBlockValidator) ValidateSyntax(ctx context.Context, blk *types.Block) error {
	return mbv.syntaxStubs[blk.Cid()]
}

// StubSyntaxValidationForBlock stubs an error when the ValidateSyntax is called
// on the with the given block.
func (mbv *StubBlockValidator) StubSyntaxValidationForBlock(blk *types.Block, err error) {
	mbv.syntaxStubs[blk.Cid()] = err
}

// StubSemanticValidationForBlock stubs an error when the ValidateSemantic is called
// on the with the given child block.
func (mbv *StubBlockValidator) StubSemanticValidationForBlock(child *types.Block, err error) {
	mbv.semanticStubs[child.Cid()] = err
}

// NewFakeProcessor creates a processor with a test validator and test rewarder
func NewFakeProcessor() *consensus.DefaultProcessor {
	return consensus.NewConfiguredProcessor(&FakeSignedMessageValidator{}, &FakeBlockRewarder{})
}

type testSigner struct{}

func (ms testSigner) SignBytes(data []byte, addr address.Address) (types.Signature, error) {
	return types.Signature{}, nil
}

// ApplyTestMessage sends a message directly to the vm, bypassing message
// validation
func ApplyTestMessage(vmState consensus.VMState, msg *types.Message) (*consensus.ApplicationResult, error) {
	return applyTestMessageWithAncestors(vmState, msg, nil)
}

// ApplyTestMessageWithActors sends a message directly to the vm with a given set of builtin actors
func ApplyTestMessageWithActors(vmState consensus.VMState, msg *types.Message) (*consensus.ApplicationResult, error) {
	return applyTestMessageWithAncestors(vmState, msg, nil)
}

// ApplyTestMessageWithGas uses the FakeBlockRewarder but the default SignedMessageValidator
func ApplyTestMessageWithGas(vmState consensus.VMState, msg *types.Message, signer *types.MockSigner,
	gasPrice types.AttoFIL, gasLimit types.GasUnits, minerOwner address.Address) (*consensus.ApplicationResult, error) {

	smsg, err := types.NewSignedMessage(*msg, signer, gasPrice, gasLimit)
	if err != nil {
		panic(err)
	}
	applier := consensus.NewConfiguredProcessor(consensus.NewDefaultMessageValidator(), consensus.NewDefaultBlockRewarder())
	return newMessageApplier(smsg, applier, vmState, minerOwner, nil)
}

func newMessageApplier(smsg *types.SignedMessage, processor *consensus.DefaultProcessor, vmState consensus.VMState,
	minerOwner address.Address, ancestors []types.TipSet) (*consensus.ApplicationResult, error) {
	amr, err := processor.ApplyMessagesAndPayRewards(context.Background(), vmState, []*types.SignedMessage{smsg}, minerOwner, ancestors)

	if len(amr.Results) > 0 {
		return amr.Results[0], err
	}

	return nil, err
}

// CreateAndApplyTestMessageFrom wraps the given parameters in a message and calls ApplyTestMessage.
func CreateAndApplyTestMessageFrom(t *testing.T, vmState consensus.VMState, from address.Address, to address.Address, val uint64, method string, ancestors []types.TipSet, params ...interface{}) (*consensus.ApplicationResult, error) {
	t.Helper()

	pdata := actor.MustConvertParams(params...)
	msg := types.NewMessage(from, to, 0, types.NewAttoFILFromFIL(val), method, pdata)
	return applyTestMessageWithAncestors(vmState, msg, ancestors)
}

// CreateAndApplyTestMessage wraps the given parameters in a message and calls
// CreateAndApplyTestMessageFrom sending the message from address.TestAddress
func CreateAndApplyTestMessage(t *testing.T, vmState consensus.VMState, to address.Address, val uint64, method string, ancestors []types.TipSet, params ...interface{}) (*consensus.ApplicationResult, error) {
	return CreateAndApplyTestMessageFrom(t, vmState, address.TestAddress, to, val, method, ancestors, params...)
}

func applyTestMessageWithAncestors(vmState consensus.VMState, msg *types.Message, ancestors []types.TipSet) (*consensus.ApplicationResult, error) {
	smsg, err := types.NewSignedMessage(*msg, testSigner{}, types.NewGasPrice(1), types.NewGasUnits(300))
	if err != nil {
		panic(err)
	}

	ta := newTestApplier()
	return newMessageApplier(smsg, ta, vmState, address.Undef, ancestors)
}

func newTestApplier() *consensus.DefaultProcessor {
	return consensus.NewConfiguredProcessor(&FakeSignedMessageValidator{}, &FakeBlockRewarder{})
}
