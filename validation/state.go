package validation

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-hamt-ipld"
	blockstore "github.com/ipfs/go-ipfs-blockstore"

	vstate "github.com/filecoin-project/chain-validation/pkg/state"

	"github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/crypto"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	wutil "github.com/filecoin-project/go-filecoin/wallet/util"
)

type StateFactory struct {
	keys *keyStore
}

var _ vstate.Factory = &StateFactory{}

func NewStateFactory() *StateFactory {
	return &StateFactory{newKeyStore()}
}

func (s *StateFactory) Signer() *keyStore {
	return s.keys
}

func (s *StateFactory) NewAddress() (vstate.Address, error) {
	return s.keys.NewAddress()
}

func (s *StateFactory) NewActor(code cid.Cid, balance vstate.AttoFIL) vstate.Actor {
	return &actorWrapper{actor.Actor{
		Code:    code,
		Balance: types.NewAttoFIL(balance),
	}}
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

//
// Key store
//

type keyStore struct {
	// Private keys by address
	keys map[address.Address]*types.KeyInfo
	// Seed for deterministic key generation.
	seed int64
}

func newKeyStore() *keyStore {
	return &keyStore{
		keys: make(map[address.Address]*types.KeyInfo),
		seed: 0,
	}
}

func (s *keyStore) NewAddress() (vstate.Address, error) {
	randSrc := rand.New(rand.NewSource(s.seed))
	prv, err := crypto.GenerateKeyFromSeed(randSrc)
	if err != nil {
		return "", err
	}

	ki := &types.KeyInfo{
		PrivateKey: prv,
		Curve:      "secp256k1",
	}
	addr, err := ki.Address()
	if err != nil {
		return "", err
	}
	s.keys[addr] = ki
	s.seed++
	return vstate.Address(addr.Bytes()), nil
}

func (as *keyStore) SignBytes(data []byte, addr address.Address) (types.Signature, error) {
	ki, ok := as.keys[addr]
	if !ok {
		return types.Signature{}, fmt.Errorf("unknown address %v", addr)
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
