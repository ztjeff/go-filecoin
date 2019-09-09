package main

import (
	"context"
	flg "flag"
	"fmt"
	"github.com/filecoin-project/go-filecoin/types"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	logging "github.com/ipfs/go-log"
	"github.com/mitchellh/go-homedir"

	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/environment"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	lpfc "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
)

var (
	network         string = "user"
	workdir         string
	binpath         string
	err             error
	fil             = 100000
	minerCount      = 5
	minerCollateral = big.NewInt(500)
	minerPrice      = big.NewFloat(0.000000000000001)
	minerExpiry     = big.NewInt(24 * 60 * 60)

	exitcode int

	flag = flg.NewFlagSet(os.Args[0], flg.ExitOnError)
)

func init() {
	logging.SetDebugLogging()

	var (
		err                error
		minerCollateralArg = minerCollateral.Text(10)
		minerPriceArg      = minerPrice.Text('f', 15)
		minerExpiryArg     = minerExpiry.Text(10)
	)

	// We default to the binary built in the project directory, fallback
	// to searching path.
	binpath, err = getFilecoinBinary()
	if err != nil {
		// Look for `go-filecoin` in the path to set `binpath` default
		// If the binary is not found, an error will be returned. If the
		// error is ErrNotFound we ignore it.
		// Error is handled after flag parsing so help can be shown without
		// erroring first
		binpath, err = exec.LookPath("go-filecoin")
		if err != nil {
			xerr, ok := err.(*exec.Error)
			if ok && xerr.Err == exec.ErrNotFound {
				err = nil
			}
		}
	}

	flag.StringVar(&network, "network", network, "set the network name to run against")
	flag.StringVar(&workdir, "workdir", workdir, "set the working directory used to store filecoin repos")
	flag.StringVar(&binpath, "binpath", binpath, "set the binary used when executing `go-filecoin` commands")
	flag.IntVar(&minerCount, "miner-count", minerCount, "number of miners")
	flag.StringVar(&minerCollateralArg, "miner-collateral", minerCollateralArg, "amount of fil each miner will use for collateral")
	flag.StringVar(&minerPriceArg, "miner-price", minerPriceArg, "price value used when creating ask for miners")
	flag.StringVar(&minerExpiryArg, "miner-expiry", minerExpiryArg, "expiry value used when creating ask for miners")

	// ExitOnError is set
	flag.Parse(os.Args[1:]) // nolint: errcheck

	// If we failed to find `go-filecoin` and it was not set, handle the error
	if len(binpath) == 0 {
		msg := "failed when checking for `go-filecoin` binary;"
		if err == nil {
			err = fmt.Errorf("no binary provided or found")
			msg = "please install or build `go-filecoin`;"
		}

		handleError(err, msg)
		os.Exit(1)
	}

	_, ok := minerCollateral.SetString(minerCollateralArg, 10)
	if !ok {
		handleError(fmt.Errorf("could not parse miner-collateral"))
		os.Exit(1)
	}

	_, ok = minerPrice.SetString(minerPriceArg)
	if !ok {
		handleError(fmt.Errorf("could not parse miner-price"))
		os.Exit(1)
	}

	_, ok = minerExpiry.SetString(minerExpiryArg, 10)
	if !ok {
		handleError(fmt.Errorf("could not parse miner-expiry"))
		os.Exit(1)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	exit := make(chan struct{}, 1)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals
		fmt.Println("Ctrl-C received, starting shutdown")
		cancel()
		exit <- struct{}{}
	}()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered from panic", r)
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
			exitcode = 1
		}
		os.Exit(exitcode)
	}()

	if len(workdir) == 0 {
		workdir, err = ioutil.TempDir("", "network-miners")
		if err != nil {
			exitcode = handleError(err)
			return
		}
	}

	if ok, err := isEmpty(workdir); !ok {
		if err == nil {
			err = fmt.Errorf("workdir is not empty: %s", workdir)
		}

		exitcode = handleError(err, "fail when checking workdir;")
		return
	}

	env, err := environment.NewDevnet(network, workdir)
	if err != nil {
		exitcode = handleError(err)
		return
	}

	// Defer the teardown, this will shuteverything down for us
	defer env.Teardown(ctx) // nolint: errcheck

	// Setup localfilecoin plugin options
	options := make(map[string]string)
	options[lpfc.AttrLogJSON] = "0"            // Disable JSON logs
	options[lpfc.AttrLogLevel] = "4"           // Set log level to Info
	options[lpfc.AttrFilecoinBinary] = binpath // Use the repo binary

	genesisURI := env.GenesisCar()
	if err != nil {
		exitcode = handleError(err, "failed to retrieve miner information from genesis;")
		return
	}

	fastenvOpts := fast.FilecoinOpts{
		InitOpts:   []fast.ProcessInitOption{fast.PODevnet(network), fast.POGenesisFile(genesisURI)},
		DaemonOpts: []fast.ProcessDaemonOption{},
	}

	// Create the processes that we will use to become miners
	var miners []*fast.Filecoin
	for i := 0; i < minerCount; i++ {
		miner, err := env.NewProcess(ctx, lpfc.PluginName, options, fastenvOpts)
		if err != nil {
			exitcode = handleError(err, "failed to create miner process;")
			return
		}

		miners = append(miners, miner)
	}

	for _, miner := range miners {
		err = series.InitAndStart(ctx, miner)
		if err != nil {
			exitcode = handleError(err, "failed series.InitAndStart;")
			return
		}

		err = env.GetFunds(ctx, miner)
		if err != nil {
			exitcode = handleError(err, "failed env.GetFunds;")
			return
		}

		pparams, err := miner.Protocol(ctx)
		if err != nil {
			exitcode = handleError(err, "failed to get protocol;")
			return
		}

		sinfo := pparams.SupportedSectors[0]

		_, err = series.CreateStorageMinerWithAsk(ctx, miner, minerCollateral, minerPrice, minerExpiry, sinfo.Size)
		if err != nil {
			exitcode = handleError(err, "failed series.CreateStorageMinerWithAsk;")
			return
		}
	}

	fmt.Println("Finished!")
	fmt.Println("Ctrl-C to exit")

	<-exit
}

func handleError(err error, msg ...string) int {
	if err == nil {
		return 0
	}

	if len(msg) != 0 {
		fmt.Println(msg[0], err)
	} else {
		fmt.Println(err)
	}

	return 1
}

// https://stackoverflow.com/a/30708914
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

func getProofsMode(smallSectors bool) types.ProofsMode {
	if smallSectors {
		return types.TestProofsMode
	}
	return types.LiveProofsMode
}

func getFilecoinBinary() (string, error) {
	gopath, err := getGoPath()
	if err != nil {
		return "", err
	}

	bin := filepath.Join(gopath, "/src/github.com/filecoin-project/go-filecoin/go-filecoin")
	_, err = os.Stat(bin)
	if err != nil {
		return "", err
	}

	if os.IsNotExist(err) {
		return "", err
	}

	return bin, nil
}

func getGoPath() (string, error) {
	gp := os.Getenv("GOPATH")
	if gp != "" {
		return gp, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "go"), nil
}
