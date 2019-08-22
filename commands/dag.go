// Package commands implements the command to print the blockchain.
package commands

import (
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"
)

var dagCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with IPLD DAG objects.",
	},
	Subcommands: map[string]*cmds.Command{
		"clear-cache": dagClearCacheCmd,
	},
}

var dagClearCacheCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Purge the cache used during transferring of piece data",
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		err := GetPorcelainAPI(env).ClearTempDatastore(req.Context)
		if err != nil {
			return err
		}

		return re.Emit("Cache cleared")
	},
}
