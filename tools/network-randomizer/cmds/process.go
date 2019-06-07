package cmds

import (
	"context"

	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"

	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	lpfc "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/constants"
)

var processCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Manage processes on the fast environment",
	},
	Subcommands: map[string]*cmds.Command{
		"add": addCmd,
		//"rm":  rmCmd,
		//"ls":  lsCmd,
	},
}

var addCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		// Setup localfilecoin plugin options
		options := make(map[string]string)
		options[lpfc.AttrLogJSON] = "0"                      // Disable JSON logs
		options[lpfc.AttrLogLevel] = "4"                     // Set log level to Info
		options[lpfc.AttrFilecoinBinary] = constants.BinPath // Use the repo binary

		fastEnv := GetPlumbingAPI(env).FastEnvironment(req.Context)

		fastenvOpts := fast.EnvironmentOpts{
			InitOpts:   []fast.ProcessInitOption{fast.POGenesisFile(fastEnv.GenesisCar())},
			DaemonOpts: []fast.ProcessDaemonOption{fast.POBlockTime(constants.BlockTime)},
		}

		genesis := fastEnv.Processes()[0]
		ctx := context.Background()

		fcnode, err := fastEnv.NewProcess(ctx, "localfilecoin", options, fastenvOpts)
		if err != nil {
			return err
		}

		if err := series.InitAndStart(ctx, fcnode); err != nil {
			return err
		}

		if err := series.Connect(ctx, genesis, fcnode); err != nil {
			return err
		}

		if err := series.SendFilecoinDefaults(ctx, genesis, fcnode, 10000); err != nil {
			return err
		}

		return nil
	},
}
