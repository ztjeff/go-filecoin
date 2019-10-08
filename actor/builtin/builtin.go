// Package builtin implements the predefined actors in Filecoin.
package builtin

import (
	"fmt"

	"github.com/filecoin-project/go-filecoin/actor/builtin/account"
	"github.com/filecoin-project/go-filecoin/actor/builtin/initactor"
	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
	"github.com/filecoin-project/go-filecoin/actor/builtin/paymentbroker"
	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/exec"
	"github.com/filecoin-project/go-filecoin/types"
	cid "github.com/ipfs/go-cid"
)

type VersionedActors struct {
	executables map[cid.Cid]exec.ExecutableActor
}

type Actors struct {
	versioned map[uint64]VersionedActors
}

// GetActorCode returns executable code for an actor by code cid at a specific protocol version
func (ba Actors) GetActorCode(code cid.Cid, version uint64) (exec.ExecutableActor, error) {
	if !code.Defined() {
		return nil, fmt.Errorf("undefined code cid")
	}
	versionedActors, ok := ba.versioned[version]
	if !ok {
		return nil, fmt.Errorf("unknown version: %d", version)
	}
	actor, ok := versionedActors.executables[code]
	if !ok {
		return nil, fmt.Errorf("unknown code: %s", code.String())
	}
	return actor, nil
}

// GetVersionedActors returns all actors applicable at a protocol version
func (ba Actors) GetVersionedActors(version uint64) (VersionedActors, error) {
	actors, ok := ba.versioned[version]
	if !ok {
		return VersionedActors{}, fmt.Errorf("unknown version: %d", version)
	}
	return actors, nil
}

type BuiltinActorsBuilder struct {
	actors Actors
}

// NewBuilder creates a builder to generate a builtin.Actor data structure
func NewBuilder() *BuiltinActorsBuilder {
	return &BuiltinActorsBuilder{actors: Actors{versioned: map[uint64]VersionedActors{}}}
}

func (bab *BuiltinActorsBuilder) AddAll(actors Actors) *BuiltinActorsBuilder {
	for version, va := range actors.versioned {
		for code, executable := range va.executables {
			bab.Add(code, version, executable)
		}
	}
	return bab
}

func (bab *BuiltinActorsBuilder) Add(code cid.Cid, version uint64, actor exec.ExecutableActor) *BuiltinActorsBuilder {
	versionedActors, ok := bab.actors.versioned[version]
	if !ok {
		versionedActors = VersionedActors{executables: map[cid.Cid]exec.ExecutableActor{}}
		bab.actors.versioned[version] = versionedActors
	}
	versionedActors.executables[code] = actor
	return bab
}

func (bab *BuiltinActorsBuilder) Build() Actors {
	return bab.actors
}

// DefaultActors is list of all actors that ship with Filecoin.
// They are indexed by their CID.
var DefaultActors = NewBuilder().
	Add(types.AccountActorCodeCid, 0, &account.Actor{}).
	Add(types.StorageMarketActorCodeCid, 0, &storagemarket.Actor{}).
	Add(types.PaymentBrokerActorCodeCid, 0, &paymentbroker.Actor{}).
	Add(types.MinerActorCodeCid, 0, &miner.Actor{}).
	Add(types.BootstrapMinerActorCodeCid, 0, &miner.Actor{Bootstrap: true}).
	Add(types.InitActorCodeCid, 0, &initactor.Actor{}).
	Build()
