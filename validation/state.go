package validation

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-hamt-ipld"
	blockstore "github.com/ipfs/go-ipfs-blockstore"

	vstate "github.com/filecoin-project/chain-validation/pkg/state"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/consensus"
	"github.com/filecoin-project/go-filecoin/crypto"
	"github.com/filecoin-project/go-filecoin/message"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	wutil "github.com/filecoin-project/go-filecoin/wallet/util"
)

type StateFactory struct {
	processor *consensus.DefaultProcessor
	addrMgr *sigAddressManager
}

var _ vstate.StateFactory = &StateFactory{}

func NewStateFactory() *StateFactory {
	// TODO change this to NewDefaultProcessor after the ApplyMessage type is changed to require
	// MeteredMessage instead of SignedMessage
	return &StateFactory{consensus.NewConfiguredProcessor(
		&message.FakeValidator{},
		consensus.NewDefaultBlockRewarder(),
		builtin.DefaultActors,
	),
		newSigAddressManager(),
	}
}

func (s *StateFactory) NewActor(code cid.Cid, balance vstate.AttoFIL) vstate.Actor {
	return &actorWrapper{actor.Actor{
		Code:    code,
		Balance: types.NewAttoFIL(balance),
	}}
}

func (s *StateFactory) NewAddress() (vstate.Address, error) {
	prv, err := crypto.GenerateKeyFromSeed(bytes.NewReader([]byte{s.addrMgr.seed}))
	if err != nil {
		return "", err
	}

	ki := &types.KeyInfo{
		PrivateKey: prv,
		Curve:     "secp256k1",
	}
	newAddr, err := vstate.NewSecp256k1Address(ki.PublicKey())
	if err != nil {
		return "", err
	}
	s.addrMgr.addressKeys[newAddr] = ki
	s.addrMgr.seed++
	return newAddr, nil
}

func (s *StateFactory) NewState(actors map[vstate.Address]vstate.Actor) (vstate.Tree, vstate.StorageMap, error) {
	ctx := context.TODO()

	bs := blockstore.NewBlockstore(datastore.NewMapDatastore())
	cst := hamt.CSTFromBstore(bs)
	treeImpl := state.NewEmptyStateTree(cst)
	storageImpl := vm.NewStorageMap(bs)

	for addr, act := range actors {
		actAddr, err := address.NewFromBytes([]byte(addr))
		if err != nil {
			return nil, nil, err
		}
		actw := act.(*actorWrapper)
		fmt.Println("setting actor", actw)
		if err := treeImpl.SetActor(ctx, actAddr, &actw.Actor); err != nil {
			return nil, nil, err
		}
	}

	_, err := treeImpl.Flush(ctx)
	if err != nil {
		return nil, nil, err
	}
	return &stateTreeWrapper{treeImpl}, &storageMapWrapper{storageImpl}, nil
}

func (s *StateFactory) ApplyMessage(tree vstate.Tree, storage vstate.StorageMap, eCtx *vstate.ExecutionContext, message interface{}) (vstate.Tree, error) {
	ctx := context.TODO()
	stateTree := tree.(*stateTreeWrapper).Tree
	vms := storage.(*storageMapWrapper).StorageMap
	msg := message.(*types.SignedMessage) // FIXME this will fail because it's not a signed message
	minerOwner, err := address.NewFromBytes([]byte(eCtx.MinerOwner))
	if err != nil {
		return nil, err
	}
	blockHeight := types.NewBlockHeight(eCtx.Epoch)
	gasTracker := vm.NewGasTracker()
	// Providing direct access to blockchain structures is very difficult and expected to go away.
	var ancestors []types.TipSet

	_, err = s.processor.ApplyMessage(ctx, stateTree, vms, msg, minerOwner, blockHeight, gasTracker, ancestors)
	if err != nil {
		return nil, err
	}
	// FIXME return the receipt value too

	// The intention of this method is to leave the input tree untouched and return a new one.
	// FIXME the state.Tree implements a mutable tree - it's impossible to hold on to the prior state
	// We might need to implement our own state tree to achieve that, or make extensive improvement to go-filecoin.
	return tree, nil
}

//
// Signed Address Manager
//

func newSigAddressManager() *sigAddressManager {
	return &sigAddressManager{
		addressKeys: make(map[vstate.Address]*types.KeyInfo),
		seed: 0,
	}
}

type sigAddressManager struct {
	// mapping of addresses to their pk/sk pairs
	addressKeys map[vstate.Address]*types.KeyInfo
	seed byte
}

func (as *sigAddressManager) SignBytes(data []byte, addr vstate.Address) (vstate.Signature, error) {
	ki, ok := as.addressKeys[addr]
	if !ok {
		return vstate.Signature{}, fmt.Errorf("unknown address for signing: %v", addr)
	}
	return wutil.Sign(ki.Key(), data)
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

//
// Storage wrapper
//

type storageMapWrapper struct {
	vm.StorageMap
}

func (s *storageMapWrapper) NewStorage(addr vstate.Address, actor vstate.Actor) (vstate.Storage, error) {
	fcActor := actor.(*actorWrapper)
	fcAddr, err := address.NewFromBytes([]byte(addr))
	if err != nil {
		return nil, err
	}
	return s.StorageMap.NewStorage(fcAddr, &fcActor.Actor), nil
}
