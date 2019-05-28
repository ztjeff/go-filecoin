package cmds

import (
	"fmt"
	"io"

	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"
	//"github.com/libp2p/go-libp2p-crypto"

	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/config"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/node"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/repo"
)

var initCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Initialize a fcnr repo",
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		newConfig := config.NewDefaultConfig()

		/*
			repoDir, _ := req.Options[OptionRepoDir].(string)
			if err := re.Emit(fmt.Sprintf("initializing filecoin node at %s\n", repoDir)); err != nil {
				return err
			}
			repoDir, err = paths.GetRepoPath(repoDir)
			if err != nil {
				return err
			}
		*/
		repoDir := "~/.fcnr"

		if err := repo.InitFSRepo(repoDir, newConfig); err != nil {
			return err
		}
		rep, err := repo.OpenFSRepo(repoDir)
		if err != nil {
			return err
		}

		// The only error Close can return is that the repo has already been closed
		defer rep.Close() // nolint: errcheck

		return node.Init(req.Context, rep)
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(initTextEncoder),
	},
}

func initTextEncoder(req *cmds.Request, w io.Writer, val interface{}) error {
	_, err := fmt.Fprintf(w, val.(string))
	return err
}
