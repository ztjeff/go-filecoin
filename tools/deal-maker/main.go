package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/filecoin-project/go-filecoin/types"
	"io"
	"io/ioutil"
	"time"

	"github.com/ipfs/go-ipfs-files"
	logging "github.com/ipfs/go-log"

	"github.com/filecoin-project/go-filecoin/porcelain"
	"github.com/filecoin-project/go-filecoin/protocol/storage/storagedeal"
	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/environment"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	lpfc "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

var network string = "staging"
var blocktime time.Duration = time.Second * 30
var max string = "0.000000000000001"

func init() {
	logging.SetDebugLogging()
}

func main() {
	ctx := context.Background()

	workdir, err := ioutil.TempDir("", "deal-maker")
	env, err := environment.NewDevnet(network, workdir)
	if err != nil {
		panic(err)
	}

	defer env.Teardown(ctx)
	ctx = series.SetCtxSleepDelay(ctx, blocktime)

	options := make(map[string]string)
	options[lpfc.AttrLogJSON] = "0"
	options[lpfc.AttrLogLevel] = "4"

	genesisURI := env.GenesisCar()
	if err != nil {
		panic(err)
	}

	fastenvOpts := fast.FilecoinOpts{
		InitOpts:   []fast.ProcessInitOption{fast.PODevnetStaging(), fast.POGenesisFile(genesisURI)},
		DaemonOpts: []fast.ProcessDaemonOption{},
	}

	node, err := env.NewProcess(ctx, lpfc.PluginName, options, fastenvOpts)
	if err != nil {
		panic(err)
	}

	err = series.InitAndStart(ctx, node)
	if err != nil {
		panic(err)
	}

	err = env.GetFunds(ctx, node)
	if err != nil {
		panic(err)
	}

	pparams, err := node.Protocol(ctx)
	if err != nil {
		panic(err)
	}

	sinfo := pparams.SupportedSectors[0]

	maxPrice, _ := types.NewAttoFILFromFILString(max)

	/////////////////////////////////////////////////////////

	for {
		dec, err := node.ClientListAsks(ctx)
		if err != nil {
			panic(err)
		}

		var asks []porcelain.Ask
		for {
			var ask porcelain.Ask

			err := dec.Decode(&ask)
			if err != nil && err != io.EOF {
				fmt.Printf("ERROR: %s\n", err)
				continue
			}

			if err == io.EOF {
				break
			}

			if !ask.Price.GreaterThan(maxPrice) {
				asks = append(asks, ask)
			}
		}

		for _, ask := range asks {
			dataReader := io.LimitReader(rand.Reader, int64(sinfo.MaxPieceSize.Uint64()))
			_, deal, err := series.ImportAndStoreWithDuration(ctx, node, ask, 256, files.NewReaderFile(dataReader))
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
				continue
			}

			_, err = series.WaitForDealState(ctx, node, deal, storagedeal.Complete)
			if err != nil {
				fmt.Printf("ERROR: %s\n", err)
				continue
			}
		}
	}
}
