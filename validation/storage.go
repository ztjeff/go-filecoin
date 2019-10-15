package validation

import (

	vstate "github.com/filecoin-project/chain-validation/pkg/state"
	blockstore "github.com/ipfs/go-ipfs-blockstore"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/vm"
)

type StorageFactory struct {
}

var _ vstate.StorageFactory = &StorageFactory{}

func (StorageFactory) NewStorageMap(store blockstore.Blockstore) vstate.StorageMap {
	return StorageMapWrapper{vm.NewStorageMap(store)}
}

//
// StorageMap Wrapper
//

type StorageMapWrapper struct {
	vm.StorageMap
}

func (s StorageMapWrapper) NewStorage(addr vstate.Address, actor vstate.Actor) vstate.Storage {
	fcActor := actor.(*actorWrapper)
	fcAddr, err := address.NewFromBytes([]byte(addr))
	if err != nil {
		// panic because returning an error breaks the interface in go-filecoin
		panic(err)
	}
	return s.StorageMap.NewStorage(fcAddr, &fcActor.Actor)
}

