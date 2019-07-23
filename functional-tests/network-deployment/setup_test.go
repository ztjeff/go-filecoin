package networkdeployment_test

import (
	"context"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	th "github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/environment"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
	"github.com/filecoin-project/go-filecoin/types"

	"github.com/ipfs/go-ipfs-files"
)

type Foo struct {
	PluginOptions map[string]string
	FastOptions   fast.FilecoinOpts
	ConfigFn      func(context.Context, *fast.Filecoin) error
	PluginName    string
	Network       string
}

func setup(t *testing.T, network string) (context.Context, environment.Environment, *Foo) {
	if network == "local" {
		return makeLocal(t)
	}

	return makeDevnet(t, network)
}

func makeDevnet(t *testing.T, network string) (context.Context, environment.Environment, *Foo) {
	ctx := context.Background()

	// Create a directory for the test using the test name (mostly for FAST)
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	// Create an environment to connect to the devnet
	env, err := environment.NewDevnet(network, dir)
	require.NoError(t, err)

	ctx = series.SetCtxSleepDelay(ctx, time.Second*30)

	// Setup options for nodes.
	pluginOpts := make(map[string]string)
	pluginOpts[localplugin.AttrLogJSON] = "0"
	pluginOpts[localplugin.AttrLogLevel] = "4"
	pluginOpts[localplugin.AttrFilecoinBinary] = th.MustGetFilecoinBinary()

	genesisURI := env.GenesisCar()
	fastOpts := fast.FilecoinOpts{
		InitOpts:   []fast.ProcessInitOption{fast.POGenesisFile(genesisURI), fast.PODevnet(network)},
		DaemonOpts: []fast.ProcessDaemonOption{},
	}

	return ctx, env, &Foo{
		PluginOptions: pluginOpts,
		FastOptions:   fastOpts,
		ConfigFn: func(ctx context.Context, node *fast.Filecoin) error {
			return nil
		},
		PluginName: localplugin.PluginName,
		Network:    network,
	}
}

func makeLocal(t *testing.T) (context.Context, environment.Environment, *Foo) {
	ctx := context.Background()

	// Create a directory for the test using the test name (mostly for FAST)
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)

	// Create an environment to connect to the devnet
	env, err := environment.NewMemoryGenesis(big.NewInt(1000000), dir, types.TestProofsMode)
	require.NoError(t, err)

	ctx = series.SetCtxSleepDelay(ctx, time.Second*5)

	// Setup options for nodes.
	pluginOpts := make(map[string]string)
	pluginOpts[localplugin.AttrLogJSON] = "0"
	pluginOpts[localplugin.AttrLogLevel] = "5"
	pluginOpts[localplugin.AttrFilecoinBinary] = th.MustGetFilecoinBinary()

	genesisURI := env.GenesisCar()
	fastOpts := fast.FilecoinOpts{
		InitOpts:   []fast.ProcessInitOption{fast.POGenesisFile(genesisURI)},
		DaemonOpts: []fast.ProcessDaemonOption{fast.POBlockTime(time.Second * 5)},
	}

	ctx = series.SetCtxSleepDelay(ctx, time.Second*5)

	genesisMiner, err := env.GenesisMiner()
	require.NoError(t, err)

	// Setup nodes used for the test
	genesis, err := env.NewProcess(ctx, localplugin.PluginName, pluginOpts, fastOpts)
	require.NoError(t, err)

	// Start setting up the nodes
	// Setup Genesis
	err = series.SetupGenesisNode(ctx, genesis, genesisMiner.Address, files.NewReaderFile(genesisMiner.Owner))
	require.NoError(t, err)

	err = genesis.MiningStart(ctx)
	require.NoError(t, err)

	details, err := genesis.ID(ctx)
	require.NoError(t, err)

	return ctx, env, &Foo{
		PluginOptions: pluginOpts,
		FastOptions:   fastOpts,
		ConfigFn: func(ctx context.Context, node *fast.Filecoin) error {
			config, err := node.Config()
			if err != nil {
				return err
			}

			config.Bootstrap.Addresses = []string{details.Addresses[0].String()}
			config.Bootstrap.MinPeerThreshold = 1
			config.Bootstrap.Period = "10s"

			return node.WriteConfig(config)
		},
		PluginName: localplugin.PluginName,
		Network:    "local",
	}
}
