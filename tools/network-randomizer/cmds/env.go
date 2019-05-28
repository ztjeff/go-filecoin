package cmds

import (
	"context"

	"github.com/ipfs/go-ipfs-cmds"
)

// Env is the environment passed to commands. Implements cmds.Environment.
type Env struct {
	ctx context.Context
}

var _ cmds.Environment = (*Env)(nil)

// Context returns the context of the environment.
func (ce *Env) Context() context.Context {
	return ce.ctx
}
