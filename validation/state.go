package validation

import (
	"context"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"

	vstate "github.com/filecoin-project/chain-validation/pkg/state"
	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/filecoin-project/go-filecoin/vm/errors"
)

type StateFactory struct {
}

var _ vstate.StateFactory = &StateFactory{}

func (StateFactory) NewActor(code cid.Cid, balance vstate.AttoFIL) vstate.Actor {
	return &actorWrapper{actor.Actor{
		Code:    code,
		Balance: types.NewAttoFIL(balance),
	}}
}

func (StateFactory) NewState(actors map[vstate.Address]vstate.Actor) (vstate.Tree, error) {
	ctx := context.TODO()

	cst := hamt.NewCborStore()
	fcTree := state.NewEmptyStateTree(cst)

	for addr, act := range actors {
		actAddr, err := address.NewFromBytes([]byte(addr))
		if err != nil {
			return nil, err
		}
		actw := act.(*actorWrapper)
		if err := fcTree.SetActor(ctx, actAddr, &actw.Actor); err != nil {
			return nil, err
		}
	}

	_, err := fcTree.Flush(ctx)
	if err != nil {
		return nil, err
	}
	return &stateTreeWrapper{fcTree}, nil
}

func (StateFactory) ApplyMessage(vmContext *vstate.VMContext, tree vstate.Tree, message interface{}) (vstate.Tree, error) {
	ctx := context.TODO()

	// get the message as a filecoin message
	fcMsg := message.(*types.Message)

	// from actor as filecoin actor
	rawActor, err := tree.Actor(vstate.Address(fcMsg.From.Bytes()))
	if err != nil {
		return nil, err
	}
	fcFromActor := rawActor.(*actorWrapper)

	// to actor as filecoin actor
	rawActor, err = tree.Actor(vstate.Address(fcMsg.To.Bytes()))
	if err != nil {
		return nil, err
	}
	fcToActor := rawActor.(*actorWrapper)

	// need a _cached_state tree
	fcTree := tree.(state.Tree)
	cachedSt := state.NewCachedStateTree(fcTree)

	// to filecoin storage map
	fcStorageMap := vmContext.Store.(*StorageMapWrapper)

	vmCtxParams := vm.NewContextParams{
		From:        &fcFromActor.Actor,
		To:          &fcToActor.Actor,
		Message:     fcMsg,
		State:       cachedSt,
		GasTracker:  vm.NewGasTracker(),
		StorageMap:  fcStorageMap.StorageMap,
		//BlockHeight:
		//Ancestors:
		//Actors:
	}
	vmCtx := vm.NewVMContext(vmCtxParams)

	// TODO this isn't write, need to check errors correctly
	_, _, vmErr := vm.Send(ctx, vmCtx)
	if vmErr != nil {
		return nil, err
	}
	// if there wasn't a vm error we can update state
	if err := cachedSt.Commit(ctx); err != nil {
		return nil, err
	}

	// At this point we consider the message successfully applied so inc
	// the nonce.
	fromActor, err := fcTree.GetActor(ctx, fcMsg.From)
	if err != nil {
		return nil, errors.FaultErrorWrap(err, "couldn't load from actor")
	}
	fromActor.IncNonce()
	if err := fcTree.SetActor(ctx, fcMsg.From, fromActor); err != nil {
		return nil, errors.FaultErrorWrap(err, "could not set from actor after inc nonce")
	}

	// in go-filecoin after a message is applied it walks all the way up the consensus stack and
	// calls  stateTree.Flush in RunStateTransition(), well pretend that's here.
	if _, err := fcTree.Flush(ctx); err != nil {
		return nil, err
	}

	stw := &stateTreeWrapper{
		Tree: fcTree,
	}

	return stw, nil

}

//
// Actor Wrapper
//

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

//
// State Tree Wrapper
//

type stateTreeWrapper struct {
	state.Tree
}

func (s *stateTreeWrapper) Actor(addr vstate.Address) (vstate.Actor, error) {
	vaddr, err := address.NewFromBytes([]byte(addr))
	if err != nil {
		return nil, err
	}
	fcActor, err := s.Tree.GetActor(context.TODO(), vaddr)
	if err != nil {
		return nil, err
	}
	return &actorWrapper{*fcActor}, nil
}

func (s *stateTreeWrapper) Cid() cid.Cid {
	panic("implement me")
}

func (s *stateTreeWrapper) ActorStorage(vstate.Address) (vstate.Storage, error) {
	panic("implement me")
}

