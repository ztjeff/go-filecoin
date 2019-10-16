package validation

import (
	"math/big"

	"github.com/filecoin-project/chain-validation/pkg/chain"
	"github.com/filecoin-project/chain-validation/pkg/state"

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
	msg := types.NewMessage(fromDec, toDec, nonce, valueDec, string(method), paramsDec)

	gasPrice:= types.NewAttoFIL(big.NewInt(1))
	gasLimit := types.NewGasUnits(1000)
	return types.NewSignedMessage(*msg, mf.signer, gasPrice, gasLimit)
}
