package cmds

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"syscall"

	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/go-ipfs-cmds/cli"
	cmdhttp "github.com/ipfs/go-ipfs-cmds/http"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multiaddr-net"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/repo"
)

const (
	// Environment variable used to see the daemons api.
	env_fcnr_api = "FCNR_API"

	// APIPrefix is the prefix for the http version of the api.
	APIPrefix = "/api"

	// OptionAPI is the name of the option for specifying the api port.
	OptionAPI = "cmdapiaddr"

	// OptionRepoDir is the name of the option for specifying the directory of the repo.
	OptionRepoDir = "repodir"
)

// command object for the local cli
var rootCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "A filecoin network randomizer",
		Subcommands: `
START RUNNING FILECOIN
  fcnr init                   - Initialize a fcnr repo
  fcnr daemon                 - Start a long-running daemon process
`,
	},
	Subcommands: make(map[string]*cmds.Command),
}

// command object for the daemon
var rootCmdDaemon = &cmds.Command{
	Subcommands: make(map[string]*cmds.Command),
}

// all top level commands, not available to daemon
var rootSubcmdsLocal = map[string]*cmds.Command{
	"daemon": daemonCmd,
	"init":   initCmd,
	//"version": versionCmd,
}

var rootSubcmdsDaemon = map[string]*cmds.Command{
	"process": processCmd,
}

func init() {
	for k, v := range rootSubcmdsLocal {
		rootCmd.Subcommands[k] = v
	}

	for k, v := range rootSubcmdsDaemon {
		rootCmd.Subcommands[k] = v
		rootCmdDaemon.Subcommands[k] = v
	}

}

// Run processes the arguments and stdin
func Run(args []string, stdin, stdout, stderr *os.File) (int, error) {
	err := cli.Run(context.Background(), rootCmd, args, stdin, stdout, stderr, buildEnv, makeExecutor)
	if err == nil {
		return 0, nil
	}
	if exerr, ok := err.(cli.ExitError); ok {
		return int(exerr), nil
	}
	return 1, err
}

func buildEnv(ctx context.Context, req *cmds.Request) (cmds.Environment, error) {
	return &Env{ctx: ctx}, nil
}

type executor struct {
	api  string
	exec cmds.Executor
}

func (e *executor) Execute(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	if e.api == "" {
		return e.exec.Execute(req, re, env)
	}

	client := cmdhttp.NewClient(e.api, cmdhttp.ClientWithAPIPrefix(APIPrefix))

	res, err := client.Send(req)
	if err != nil {
		if isConnectionRefused(err) {
			return cmdkit.Errorf(cmdkit.ErrFatal, "Connection Refused. Is the daemon running?")
		}
		return cmdkit.Errorf(cmdkit.ErrFatal, err.Error())
	}

	// copy received result into cli emitter
	err = cmds.Copy(re, res)
	if err != nil {
		return cmdkit.Errorf(cmdkit.ErrFatal|cmdkit.ErrNormal, err.Error())
	}
	return nil
}

func makeExecutor(req *cmds.Request, env interface{}) (cmds.Executor, error) {
	isDaemonRequired := requiresDaemon(req)
	var api string
	if isDaemonRequired {
		var err error
		api, err = getAPIAddress(req)
		if err != nil {
			return nil, err
		}
	}

	if api == "" && isDaemonRequired {
		return nil, ErrMissingDaemon
	}

	return &executor{
		api:  api,
		exec: cmds.NewExecutor(rootCmd),
	}, nil
}

func requiresDaemon(req *cmds.Request) bool {
	for _, cmd := range rootSubcmdsLocal {
		if req.Command == cmd {
			return false
		}
	}
	return true
}

func getAPIAddress(req *cmds.Request) (string, error) {
	var rawAddr string
	var err error
	// second highest precedence is env vars.
	if envapi := os.Getenv(env_fcnr_api); envapi != "" {
		rawAddr = envapi
	}

	// first highest precedence is cmd flag.
	if apiAddress, ok := req.Options[OptionAPI].(string); ok && apiAddress != "" {
		rawAddr = apiAddress
	}

	// we will read the api file if no other option is given.
	if len(rawAddr) == 0 {
		/*
			repoDir, _ := req.Options[OptionRepoDir].(string)
				repoDir, err = paths.GetRepoPath(repoDir)
				if err != nil {
					return "", err
				}
		*/
		repoDir := "~/.fcnr"
		rawAddr, err = repo.APIAddrFromRepoPath(repoDir)
		if err != nil {
			return "", errors.Wrap(err, "can't find API endpoint address in environment, command-line, or local repo (is the daemon running?)")
		}
	}

	maddr, err := ma.NewMultiaddr(rawAddr)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("unable to convert API endpoint address %s to a multiaddr", rawAddr))
	}

	_, host, err := manet.DialArgs(maddr)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("unable to dial API endpoint address %s", maddr))
	}

	return host, nil
}

func isConnectionRefused(err error) bool {
	urlErr, ok := err.(*url.Error)
	if !ok {
		return false
	}

	opErr, ok := urlErr.Err.(*net.OpError)
	if !ok {
		return false
	}

	syscallErr, ok := opErr.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	return syscallErr.Err == syscall.ECONNREFUSED
}
