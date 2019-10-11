package consensus

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/actor/builtin"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/exec"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/version"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/pkg/errors"
)

var (
	// ErrNoMethod is returned by Get when there is no method signature (eg, transfer).
	ErrNoMethod = errors.New("no method")
	// ErrNoActorImpl is returned by Get when the actor implementation doesn't exist, eg
	// the actor address is an empty actor, an address that has received a transfer of FIL
	// but hasn't yet been upgraded to an account actor. (The actor implementation might
	// also genuinely be missing, which is not expected.)
	ErrNoActorImpl = errors.New("no actor implementation")
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
	versionTable *version.ProtocolVersionTable
}

// NewVMStateStore creates a VMStateStore
func NewVMStateStore(cst *hamt.CborIpldStore, bs bstore.Blockstore, actors builtin.Actors, pvt *version.ProtocolVersionTable) VMStateStore {
	return VMStateStore{
		ipldStore:    cst,
		blockstore:   bs,
		actors:       actors,
		versionTable: pvt,
	}
}

func (vss VMStateStore) State(ctx context.Context, stateRoot cid.Cid, bh *types.BlockHeight) (VMState, error) {
	// create state tree (empty if cid is undefined)
	var st state.Tree
	var err error
	if stateRoot.Defined() {
		st, err = state.LoadStateTree(ctx, vss.ipldStore, stateRoot)
		if err != nil {
			return VMState{}, err
		}
	} else {
		st = state.NewEmptyStateTree(vss.ipldStore)
	}

	// find actors for current block height
	version, err := vss.versionTable.VersionAt(bh)
	if err != nil {
		return VMState{}, nil
	}
	actors, err := vss.actors.GetVersionedActors(version)
	if err != nil {
		return VMState{}, err
	}

	return VMState{
		stateRoot:   stateRoot,
		st:          st,
		storage:     vm.NewStorageMap(vss.blockstore),
		actors:      actors,
		blockHeight: bh,
	}, nil
}

// VMState represents the state of the VM at a specific state transition
type VMState struct {
	stateRoot   cid.Cid
	st          state.Tree
	storage     vm.StorageMap
	actors      builtin.VersionedActors
	blockHeight *types.BlockHeight
}

func (vs VMState) Tree() state.Tree {
	return vs.st
}

func (vs VMState) CachedTree() *state.CachedTree {
	return state.NewCachedStateTree(vs.st)
}

func (vs VMState) Storage() vm.StorageMap {
	return vs.storage
}

func (vs VMState) Actors() builtin.VersionedActors {
	return vs.actors
}

func (vs VMState) FlushTree(ctx context.Context) (cid.Cid, error) {
	return vs.st.Flush(ctx)
}

func (vs VMState) StateRoot() cid.Cid {
	return vs.stateRoot
}

func (vs VMState) BlockHeight() *types.BlockHeight {
	return vs.blockHeight
}

// Query sends a read-only message against the state of the snapshot.
func (vs VMState) Query(ctx context.Context, optFrom, to address.Address, method string, params ...interface{}) ([][]byte, error) {
	encodedParams, err := abi.ToEncodedValues(params...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message params")
	}

	r, ec, err := NewDefaultProcessor().CallQueryMethod(ctx, vs, to, method, encodedParams, optFrom)
	if err != nil {
		return nil, errors.Wrap(err, "query method returned an error")
	} else if ec != 0 {
		return nil, errors.Errorf("query method returned a non-zero error code %d", ec)
	}
	return r, nil
}

// GetActorSignature returns the signature of the given actor's given method.
// The function signature is typically used to enable a caller to decode the
// output of an actor method call (message).
func (vs VMState) GetActorSignature(ctx context.Context, actorAddr address.Address, method string) (*exec.FunctionSignature, error) {
	if method == "" {
		return nil, ErrNoMethod
	}

	actor, err := vs.Tree().GetActor(ctx, actorAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get actor")
	} else if actor.Empty() {
		return nil, ErrNoActorImpl
	}

	executable, err := vs.actors.GetActorCode(actor.Code)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load actor code")
	}

	export, ok := executable.Exports()[method]
	if !ok {
		return nil, fmt.Errorf("missing export: %s", method)
	}

	return export, nil
}
