package validation

import (
	vstate "github.com/filecoin-project/chain-validation/pkg/state"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
)

type StateFactory struct {
}

var _ vstate.Factory = &StateFactory{}

func (StateFactory) NewActor(code cid.Cid, balance vstate.AttoFIL) vstate.Actor {
	return &actorWrapper{actor.Actor{
		Code:    code,
		Balance: types.NewAttoFIL(balance),
	}}
}

func (StateFactory) NewState(actors map[vstate.Address]vstate.Actor) vstate.Tree {
	state.NewEmptyStateTree()
	panic("implement me")
}

func (StateFactory) ApplyMessage(vstate.Tree, interface{}) vstate.Tree {
	panic("implement me")
}

type actorWrapper struct {
	actor.Actor
}

func (a *actorWrapper) Code() cid.Cid {
	return a.Actor.Code
}

func (a *actorWrapper) Head() cid.Cid {
	return a.Actor.Head
}

func (a *actorWrapper) Nonce() uint64 {
	return uint64(a.Actor.Nonce)
}

func (a *actorWrapper) Balance() vstate.AttoFIL {
	return a.Actor.Balance.AsBigInt()
}
