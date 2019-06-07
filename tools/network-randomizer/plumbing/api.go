package plumbing

import (
	"context"
	"github.com/filecoin-project/go-filecoin/tools/fast"
)

type API struct {
	fastEnv fast.Environment
}

type APIDeps struct {
	FastEnv fast.Environment
}

func New(deps *APIDeps) *API {
	return &API{
		fastEnv: deps.FastEnv,
	}
}

func (api *API) FastEnvironment(ctx context.Context) fast.Environment {
	return api.fastEnv
}
