package cmds

import (
	"github.com/pkg/errors"
)

var (
	// ErrMissingDaemon is the error returned when trying to execute a command that requires the daemon to be started.
	ErrMissingDaemon = errors.New("daemon must be started before using this command")
)
