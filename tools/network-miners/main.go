package main

import (
	"context"
	"crypto/rand"
	"flag"
	"io"
	"log"
	"math/big"
	"os/exec"
	"time"

	"github.com/ipfs/go-ipfs-files"

	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/environment"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

func main() {
	var workdir = "/storage/nodes"
	var binpath = ""
	var minercount = 5
	var network = "staging"
	var sealperiod = 10
	var err error

	binpath, err = exec.LookPath("go-filecoin")
	if err != nil {
		xerr, ok := err.(*exec.Error)
		if ok && xerr.Err == exec.ErrNotFound {
			err = nil
		}
	}

	flag.StringVar(&workdir, "workdir", workdir, "set the base directory for node repos")
	flag.StringVar(&binpath, "binpath", binpath, "set the binary used when executing `go-filecoin` commands")
	flag.StringVar(&network, "network", network, "set the network to setup miners for")
	flag.IntVar(&minercount, "miner-count", minercount, "number of miners")
	flag.IntVar(&sealperiod, "seal-period", sealperiod, "number of minutes between sealed sectors")

	flag.Parse()

	if len(binpath) == 0 {
		log.Fatal("binpath is empty, please specify a path to the go-filecoin binary or place it in your path")
	}

	ctx := context.Background()

	env, err := environment.NewDevnet(network, workdir)
	if err != nil {
		log.Fatal(err)
	}

	defer env.Teardown(ctx) // nolint: errcheck

	// Setup options for nodes.
	options := make(map[string]string)
	options[localplugin.AttrLogJSON] = "0"
	options[localplugin.AttrLogLevel] = "4"
	options[localplugin.AttrFilecoinBinary] = binpath

	genesisURI := env.GenesisCar()

	fastenvOpts := fast.FilecoinOpts{
		InitOpts:   []fast.ProcessInitOption{fast.POGenesisFile(genesisURI), fast.PODevnet(network)},
		DaemonOpts: []fast.ProcessDaemonOption{},
	}

	var miners []*fast.Filecoin
	for i := 0; i < minercount; i++ {
		miner, err := env.NewProcess(ctx, localplugin.PluginName, options, fastenvOpts)
		if err != nil {
			log.Fatal(err)
		}

		miners = append(miners, miner)
	}

	for _, miner := range miners {
		log.Printf("Starting %s", miner.Dir())
		err = series.InitAndStart(ctx, miner)
		if err != nil {
			log.Fatal(err)
		}
	}

	time.Sleep(time.Minute)

	collateral := big.NewInt(32)

	for _, miner := range miners {
		log.Printf("Getting funds for %s", miner.Dir())
		err := env.GetFunds(ctx, miner)
		if err != nil {
			log.Fatal(err)
		}

		pparams, err := miner.Protocol(ctx)
		if err != nil {
			log.Fatal(err)
			return
		}

		sinfo := pparams.SupportedSectors[0]

		log.Printf("Creating miner for %s", miner.Dir())
		_, err = miner.MinerCreate(ctx, collateral, fast.AOSectorSize(sinfo.Size), fast.AOPrice(big.NewFloat(0.0001)), fast.AOLimit(300))
		if err != nil {
			log.Fatal(err)
		}
	}

	ticker := time.Tick(time.Duration(sealperiod) * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			for _, miner := range miners {
				log.Printf("Entering new sealing interval for miner %s", miner.Dir())
				var err error

				pparams, err := miner.Protocol(ctx)
				if err != nil {
					log.Printf("Failed to lookup protocol info: %s", err)
					continue
				}

				sinfo := pparams.SupportedSectors[0]

				dataReader := io.LimitReader(rand.Reader, int64(sinfo.MaxPieceSize.Uint64()))

				_, err = miner.AddPiece(ctx, files.NewReaderFile(dataReader))
				if err != nil {
					log.Printf("Failed to add piece: %s", err)
					continue
				}

				err = miner.SealNow(ctx)
				if err != nil {
					log.Printf("Failed to seal: %s", err)
					continue
				}
			}
		}
	}
}
