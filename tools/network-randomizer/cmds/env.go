package cmds

import (
	"context"

	"github.com/ipfs/go-ipfs-cmds"

	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/plumbing"
)

// Env is the environment passed to commands. Implements cmds.Environment.
type Env struct {
	ctx         context.Context
	plumbingAPI *plumbing.API
}

var _ cmds.Environment = (*Env)(nil)

// Context returns the context of the environment.
func (ce *Env) Context() context.Context {
	return ce.ctx
}

func GetPlumbingAPI(env cmds.Environment) *plumbing.API {
	ce := env.(*Env)
	return ce.plumbingAPI
}
