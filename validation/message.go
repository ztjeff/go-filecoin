package validation

import (
	"github.com/filecoin-project/chain-validation/pkg/chain"
	"github.com/filecoin-project/chain-validation/pkg/state"

	"github.com/filecoin-project/go-filecoin/abi"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
)

type MessageFactory struct {
}

var _ chain.MessageFactory = &MessageFactory{}

func NewMessageFactory() *MessageFactory {
	return &MessageFactory{}
}

func (mf *MessageFactory) MakeMessage(from, to state.Address, method state.MethodID, nonce uint64, value state.AttoFIL, params ...interface{}) (interface{}, error) {
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
	return types.NewMessage(fromDec, toDec, nonce, valueDec, string(method), paramsDec), nil
}
