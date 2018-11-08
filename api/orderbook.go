package api

import (
	"context"

	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
)

// Orderbook is the interface that defines methods to interact with the orderbook.
type Orderbook interface {
	Asks(ctx context.Context) (storagemarket.AskSet, error)
}
