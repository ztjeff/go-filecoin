package commands

import (
	"encoding/json"
	cmds "gx/ipfs/QmPTfgFTo9PFr1PvPKyKoeMgBvYPh6cX3aDP7DHKVbnCbi/go-ipfs-cmds"
	cmdkit "gx/ipfs/QmSP88ryZkHSRn1fnngAaV2Vcn63WUJzAavnRM9CVdU1Ky/go-ipfs-cmdkit"
	"io"

	"github.com/filecoin-project/go-filecoin/actor/builtin/storagemarket"
)

var orderbookCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with the order book",
	},
	Subcommands: map[string]*cmds.Command{
		"asks": askCmd,
	},
}

var askCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) {
		asks, err := GetAPI(env).Orderbook().Asks(req.Context)
		if err != nil {
			re.SetError(err, cmdkit.ErrNormal)
			return
		}
		for _, ask := range asks {
			re.Emit(ask) // nolint errcheck
		}
	},
	Type: &storagemarket.Ask{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, ask *storagemarket.Ask) error {
			b, err := json.Marshal(ask)
			if err != nil {
				return err
			}
			_, err = w.Write(b)
			return err
		}),
	},
}
