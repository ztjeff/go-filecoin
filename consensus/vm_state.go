package consensus

import (
	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/version"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/ipfs/go-hamt-ipld"
	bstore "github.com/ipfs/go-ipfs-blockstore"
)

// VMStateStore manages all state that can be accessed from within actor code.
// This includes the actor code itself, the actor models (state tree), and the state
// created and managed by the actors. VMStateStore permits access to VMState structs which
// represent state at a specific state transition.
type VMStateStore struct {
	// To load the tree for the head tipset state root.
	ipldStore *hamt.CborIpldStore
	// For vm storage.
	blockstore bstore.Blockstore
	// for executable code
	actors builtin.Actors
	// to determine which code to use
	versionTable version.ProtocolVersionTable
}

// NewVMStateStore creates a VMStateStore
func NewVMStateStore(cst *hamt.CborIpldStore, bs bstore.Blockstore, actors builtin.Actors, pvt version.ProtocolVersionTable) VMStateStore {
	return VMStateStore{
		ipldStore:    cst,
		blockstore:   bs,
		actors:       actors,
		versionTable: pvt,
	}
}

// VMState represents the state of the VM at a specific state transition
type VMState struct {
	st      state.Tree
	storage vm.StorageMap
	actors  builtin.Actors
}
