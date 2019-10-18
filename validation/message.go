package validation

import (
	"github.com/filecoin-project/chain-validation/pkg/chain"
	"github.com/filecoin-project/chain-validation/pkg/state"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
)

type MessageFactory struct {
	signer types.Signer
}

var _ chain.MessageFactory = &MessageFactory{}

func NewMessageFactory(signer types.Signer) *MessageFactory {
	return &MessageFactory{signer}
}

func (mf *MessageFactory) MakeMessage(from, to state.Address, method chain.MethodID, nonce uint64,
	value, gasPrice state.AttoFIL, gasUnit state.GasUnit, params ...interface{}) (interface{}, error) {
	fromDec, err := address.NewFromBytes([]byte(from))
	if err != nil {
		return nil, err
	}
	toDec, err := address.NewFromBytes([]byte(to))
	if err != nil {
		return nil, err
	}
	valueDec := types.NewAttoFIL(value)
	paramsDec, err := abi.ToEncodedValues(params)
	if err != nil {
		return nil, err
	}
	if int(method) >= len(methods) {
		return nil, errors.Errorf("No method name for method %v", method)
	}
	methodName := methods[method]
	msg := types.NewMessage(fromDec, toDec, nonce, valueDec, methodName, paramsDec)

	return types.NewSignedMessage(*msg, mf.signer, types.NewAttoFIL(gasPrice), types.NewGasUnits(uint64(gasUnit)))
}

// Maps method enumeration values to method names.
// This will change to a mapping to method ids when method dispatch is updated to use integers.
var methods = []string{
	chain.NoMethod: "",
}
