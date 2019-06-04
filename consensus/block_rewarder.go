package consensus

import (
	"context"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/filecoin-project/go-filecoin/vm/errors"
)

// BlockRewarder applies all rewards due to the miner's owner for processing a block including block reward and gas
type BlockRewarder interface {
	// BlockReward pays out the mining reward
	BlockReward(ctx context.Context, st state.Tree, minerOwnerAddr address.Address) error

	// GasReward pays gas from the sender to the miner
	GasReward(ctx context.Context, st state.Tree, minerOwnerAddr address.Address, msg *types.SignedMessage, cost *types.AttoFIL) error
}

// DefaultBlockRewarder pays the block reward from the network actor to the miner's owner.
type DefaultBlockRewarder struct{}

// NewDefaultBlockRewarder creates a new rewarder that actually pays the appropriate rewards.
func NewDefaultBlockRewarder() *DefaultBlockRewarder {
	return &DefaultBlockRewarder{}
}

var _ BlockRewarder = (*DefaultBlockRewarder)(nil)

// BlockReward transfers the block reward from the network actor to the miner's owner.
func (br *DefaultBlockRewarder) BlockReward(ctx context.Context, st state.Tree, minerOwnerAddr address.Address) error {
	cachedTree := state.NewCachedStateTree(st)
	if err := rewardTransfer(ctx, address.NetworkAddress, minerOwnerAddr, br.BlockRewardAmount(), cachedTree); err != nil {
		return errors.FaultErrorWrap(err, "Error attempting to pay block reward")
	}
	return cachedTree.Commit(ctx)
}

// GasReward transfers the gas cost reward from the sender actor to the minerOwnerAddr
func (br *DefaultBlockRewarder) GasReward(ctx context.Context, st state.Tree, minerOwnerAddr address.Address, msg *types.SignedMessage, gas *types.AttoFIL) error {
	cachedTree := state.NewCachedStateTree(st)
	if err := rewardTransfer(ctx, msg.From, minerOwnerAddr, gas, cachedTree); err != nil {
		return errors.FaultErrorWrap(err, "Error attempting to pay gas reward")
	}
	return cachedTree.Commit(ctx)
}

// BlockRewardAmount returns the max FIL value miners can claim as the block reward.
// TODO this is one of the system parameters that should be configured as part of
// https://github.com/filecoin-project/go-filecoin/issues/884.
func (br *DefaultBlockRewarder) BlockRewardAmount() *types.AttoFIL {
	return types.NewAttoFILFromFIL(1000)
}

// rewardTransfer retrieves two actors from the given addresses and attempts to transfer the given value from the balance of the first's to the second.
func rewardTransfer(ctx context.Context, fromAddr, toAddr address.Address, value *types.AttoFIL, st *state.CachedTree) error {
	fromActor, err := st.GetActor(ctx, fromAddr)
	if err != nil {
		return errors.FaultErrorWrap(err, "could not retrieve from actor for reward transfer.")
	}

	toActor, err := st.GetOrCreateActor(ctx, toAddr, func() (*actor.Actor, error) {
		return &actor.Actor{}, nil
	})
	if err != nil {
		return errors.FaultErrorWrap(err, "failed to get To actor")
	}

	return vm.Transfer(fromActor, toActor, value)
}
