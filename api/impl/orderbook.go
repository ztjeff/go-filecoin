package impl

import (
	"context"

	cbor "gx/ipfs/QmV6BQ6fFCf9eFHDuRxvguvqfKLZtZrxthgZvDfRCs4tMN/go-ipld-cbor"

	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
	"github.com/filecoin-project/go-filecoin/address"
)

type nodeOrderbook struct {
	api *nodeAPI
}

func newNodeOrderbook(api *nodeAPI) *nodeOrderbook {
	return &nodeOrderbook{api: api}
}

func (api *nodeOrderbook) Asks(ctx context.Context) (storagemarket.AskSet, error) {
	bytes, _, err := api.api.Message().Query(
		ctx,
		address.Address{},
		address.StorageMarketAddress,
		"getAllAsks",
	)
	if err != nil {
		return nil, err
	}

	askSet := storagemarket.AskSet{}
	if err = cbor.DecodeInto(bytes[0], &askSet); err != nil {
		return nil, err
	}
	return askSet, nil
}
