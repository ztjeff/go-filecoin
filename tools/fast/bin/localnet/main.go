package main

// localnet
//
// localnet is a FAST binary tool that quickly, and easily, sets up a local network
// on the users computer. The network will stay standing till the program is closed.

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gx/ipfs/QmQmhotPUzVrMEWNK3x1R5jQ5ZHWyL7tVUrmRPjrBrvyCb/go-ipfs-files"
	bstore "gx/ipfs/QmRu7tiRnFk9mMPpVECQTBQJqXtmG132jJxA1w9A7TtpBz/go-ipfs-blockstore"
	"gx/ipfs/QmSz8kAe2JCKp2dWSG8gHSWnwSmne8YfRXTeK5HBmc9L7t/go-ipfs-exchange-offline"
	bserv "gx/ipfs/QmZsGVGCqMCNzHLNMB6q4F6yyvomqf1VxwhJwSfgo1NGaF/go-blockservice"
	logging "gx/ipfs/QmbkT7eMTyXfpeyB3ZMxxcxg7XH8t6uXp49jqzz4HB7BGF/go-log"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/proofs"
	"github.com/filecoin-project/go-filecoin/proofs/sectorbuilder"
	"github.com/filecoin-project/go-filecoin/protocol/storage/storagedeal"
	"github.com/filecoin-project/go-filecoin/repo"
	"github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	lpfc "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

var (
	workdir      string
	shell        bool
	count        = 5
	blocktime    = 5 * time.Second
	err          error
	fil          = 100000
	balance      big.Int
	smallSectors = true

	sectorSize uint64
)

func init() {
	var err error

	logging.SetDebugLogging()

	flag.StringVar(&workdir, "workdir", workdir, "set the working directory")
	flag.BoolVar(&shell, "shell", shell, "drop into a shell")
	flag.BoolVar(&smallSectors, "small-sectors", smallSectors, "enables small sectors")
	flag.IntVar(&count, "count", count, "number of miners")
	flag.DurationVar(&blocktime, "blocktime", blocktime, "duration for blocktime")

	flag.Parse()

	// Set the series global sleep delay to our blocktime
	series.GlobalSleepDelay = blocktime

	sectorSize, err = getSectorSize(smallSectors)
	if err != nil {
		handleError(err)
		os.Exit(1)
	}

	// Set the initial balance
	balance.SetInt64(int64(100 * fil))
}

func main() {
	ctx := context.Background()

	if len(workdir) == 0 {
		workdir, err = ioutil.TempDir("", "localnet")
		if err != nil {
			handleError(err)
			os.Exit(1)
		}
	}

	if ok, err := isEmpty(workdir); !ok {
		handleError(err, "workdir is not empty;")
		os.Exit(1)
	}

	env, err := fast.NewEnvironmentMemoryGenesis(&balance, workdir)
	if err != nil {
		handleError(err)
		os.Exit(1)
	}

	// Defer the teardown, this will shuteverything down for us
	defer env.Teardown(ctx) // nolint: errcheck

	binpath, err := testhelpers.GetFilecoinBinary()
	if err != nil {
		handleError(err, "no binary was found, please build go-filecoin;")
		os.Exit(1)
	}

	// Setup localfilecoin plugin options
	options := make(map[string]string)
	options[lpfc.AttrLogJSON] = "0"                                     // Disable JSON logs
	options[lpfc.AttrLogLevel] = "4"                                    // Set log level to Info
	options[lpfc.AttrUseSmallSectors] = fmt.Sprintf("%t", smallSectors) // Enable small sectors
	options[lpfc.AttrFilecoinBinary] = binpath                          // Use the repo binary

	genesisURI := env.GenesisCar()
	genesisMiner, err := env.GenesisMiner()
	if err != nil {
		handleError(err, "failed to retrieve miner information from genesis;")
		os.Exit(1)
	}

	fastenvOpts := fast.EnvironmentOpts{
		InitOpts:   []fast.ProcessInitOption{fast.POGenesisFile(genesisURI)},
		DaemonOpts: []fast.ProcessDaemonOption{fast.POBlockTime(series.GlobalSleepDelay)},
	}

	// The genesis process is the filecoin node that loads the miner that is
	// define with power in the genesis block, and the prefunnded wallet
	genesis, err := env.NewProcess(ctx, lpfc.PluginName, options, fastenvOpts)
	if err != nil {
		handleError(err, "failed to create genesis process;")
		os.Exit(1)
	}

	err = series.SetupGenesisNode(ctx, genesis, genesisMiner.Address, files.NewReaderFile(genesisMiner.Owner))
	if err != nil {
		handleError(err, "failed series.SetupGenesisNode;")
		os.Exit(1)
	}

	// Create the processes that we will use to become miners
	var miners []*fast.Filecoin
	for i := 0; i < count; i++ {
		miner, err := env.NewProcess(ctx, lpfc.PluginName, options, fastenvOpts)
		if err != nil {
			handleError(err, "failed to create miner process;")
			os.Exit(1)
		}

		miners = append(miners, miner)
	}

	// We will now go through the process of creating miners
	// InitAndStart
	// 1. Initialize node
	// 2. Start daemon
	//
	// Connect
	// 3. Connect to genesis
	//
	// SendFilecoinDefaults
	// 4. Issue FIL to node
	//
	// CreateMinerWithAsk
	// 5. Create a new miner
	// 6. Set the miner price, and get ask
	//
	// ImportAndStore
	// 7. Generated some random data and import it to genesis
	// 8. Genesis propposes a storage deal with miner
	//
	// WaitForDealState
	// 9. Query deal till posted

	var deals []*storagedeal.Response

	for _, miner := range miners {
		err = series.InitAndStart(ctx, miner)
		if err != nil {
			handleError(err, "failed series.InitAndStart;")
			os.Exit(1)
		}

		err = series.Connect(ctx, genesis, miner)
		if err != nil {
			handleError(err, "failed series.Connect;")
			os.Exit(1)
		}

		err = series.SendFilecoinDefaults(ctx, genesis, miner, fil)
		if err != nil {
			handleError(err, "failed series.SendFilecoinDefaults;")
			os.Exit(1)
		}

		pledge := uint64(10)                    // sectors
		collateral := big.NewInt(500)           // FIL
		price := big.NewFloat(0.000000001)      // price per byte/block
		expiry := big.NewInt(24 * 60 * 60 / 30) // ~24 hours

		ask, err := series.CreateMinerWithAsk(ctx, miner, pledge, collateral, price, expiry)
		if err != nil {
			handleError(err, "failed series.CreateMinerWithAsk;")
			os.Exit(1)
		}

		var data bytes.Buffer
		dataReader := io.LimitReader(rand.Reader, int64(sectorSize))
		dataReader = io.TeeReader(dataReader, &data)
		_, deal, err := series.ImportAndStore(ctx, genesis, ask, files.NewReaderFile(dataReader))
		if err != nil {
			handleError(err, "failed series.ImportAndStore;")
			os.Exit(1)
		}

		deals = append(deals, deal)

	}

	for _, deal := range deals {
		err = series.WaitForDealState(ctx, genesis, deal, storagedeal.Posted)
		if err != nil {
			handleError(err, "failed series.WaitForDealState;")
			os.Exit(1)
		}
	}

	if shell {
		client, err := env.NewProcess(ctx, lpfc.PluginName, options, fastenvOpts)
		if err != nil {
			handleError(err, "failed to create client process;")
			os.Exit(1)
		}

		err = series.InitAndStart(ctx, client)
		if err != nil {
			handleError(err, "failed series.InitAndStart;")
			os.Exit(1)
		}

		err = series.Connect(ctx, genesis, client)
		if err != nil {
			handleError(err, "failed series.Connect;")
			os.Exit(1)
		}

		err = series.SendFilecoinDefaults(ctx, genesis, client, fil)
		if err != nil {
			handleError(err, "failed series.SendFilecoinDefaults;")
			os.Exit(1)
		}

		interval, err := client.StartLogCapture()
		if err != nil {
			handleError(err, "failed to start log capture;")
			os.Exit(1)
		}

		if err := client.Shell(); err != nil {
			handleError(err, "failed to run client shell;")
			os.Exit(1)
		}

		interval.Stop()
		fmt.Println("===================================")
		fmt.Println("===================================")
		io.Copy(os.Stdout, interval) // nolint: errcheck
		fmt.Println("===================================")
		fmt.Println("===================================")
	}

	fmt.Println("Finished!")
	fmt.Println("Ctrl-C to handleError")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	<-signals
}

func handleError(err error, msg ...string) {
	if err == nil {
		return
	}

	if len(msg) != 0 {
		fmt.Println(msg[0], err)
	} else {
		fmt.Println(err)
	}
}

// https://stackoverflow.com/a/3070891://stackoverflow.com/a/30708914
func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close() // nolint: errcheck

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func getSectorSize(smallSectors bool) (uint64, error) {
	rp := repo.NewInMemoryRepo()
	blockstore := bstore.NewBlockstore(rp.Datastore())
	blockservice := bserv.New(blockstore, offline.Exchange(blockstore))

	var sectorStoreType proofs.SectorStoreType

	if smallSectors {
		sectorStoreType = proofs.Test
	} else {
		sectorStoreType = proofs.Live
	}

	cfg := sectorbuilder.RustSectorBuilderConfig{
		BlockService:     blockservice,
		LastUsedSectorID: 0,
		MetadataDir:      "",
		MinerAddr:        address.Address{},
		SealedSectorDir:  "",
		SectorStoreType:  sectorStoreType,
		StagedSectorDir:  "",
	}

	sb, err := sectorbuilder.NewRustSectorBuilder(cfg)
	if err != nil {
		return 0, err
	}

	return sb.GetMaxUserBytesPerStagedSector()
}
