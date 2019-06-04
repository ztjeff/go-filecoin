package consensus

import (
	"context"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"
	"github.com/filecoin-project/go-filecoin/vm/errors"
)

// CallQueryMethod calls a method on an actor in the given state tree. It does
// not make any changes to the state/blockchain and is useful for interrogating
// actor state. Block height bh is optional; some methods will ignore it.
func CallQueryMethod(ctx context.Context, st state.Tree, vms vm.StorageMap, to address.Address, method string, params []byte, from address.Address, optBh *types.BlockHeight) ([][]byte, uint8, error) {
	toActor, err := st.GetActor(ctx, to)
	if err != nil {
		return nil, 1, errors.ApplyErrorPermanentWrapf(err, "failed to get To actor")
	}

	// not committing or flushing storage structures guarantees changes won't make it to stored state tree or datastore
	cachedSt := state.NewCachedStateTree(st)

	msg := &types.Message{
		From:   from,
		To:     to,
		Nonce:  0,
		Value:  nil,
		Method: method,
		Params: params,
	}

	// Set the gas limit to the max because this message send should always succeed; it doesn't cost gas.
	gasTracker := vm.NewGasTracker()
	gasTracker.MsgGasLimit = types.BlockGasLimit

	vmCtxParams := vm.NewContextParams{
		To:          toActor,
		Message:     msg,
		State:       cachedSt,
		StorageMap:  vms,
		GasTracker:  gasTracker,
		BlockHeight: optBh,
	}

	vmCtx := vm.NewVMContext(vmCtxParams)
	ret, retCode, err := vm.Send(ctx, vmCtx)
	return ret, retCode, err
}

// PreviewQueryMethod estimates the amount of gas that will be used by a method
// call. It accepts all the same arguments as CallQueryMethod.
func PreviewQueryMethod(ctx context.Context, st state.Tree, vms vm.StorageMap, to address.Address, method string, params []byte, from address.Address, optBh *types.BlockHeight) (types.GasUnits, error) {
	toActor, err := st.GetActor(ctx, to)
	if err != nil {
		return types.NewGasUnits(0), errors.ApplyErrorPermanentWrapf(err, "failed to get To actor")
	}

	// not committing or flushing storage structures guarantees changes won't make it to stored state tree or datastore
	cachedSt := state.NewCachedStateTree(st)

	msg := &types.Message{
		From:   from,
		To:     to,
		Nonce:  0,
		Value:  nil,
		Method: method,
		Params: params,
	}

	// Set the gas limit to the max because this message send should always succeed; it doesn't cost gas.
	gasTracker := vm.NewGasTracker()
	gasTracker.MsgGasLimit = types.BlockGasLimit

	vmCtxParams := vm.NewContextParams{
		To:          toActor,
		Message:     msg,
		State:       cachedSt,
		StorageMap:  vms,
		GasTracker:  gasTracker,
		BlockHeight: optBh,
	}
	vmCtx := vm.NewVMContext(vmCtxParams)
	_, _, err = vm.Send(ctx, vmCtx)

	return vmCtx.GasUnits(), err
}
