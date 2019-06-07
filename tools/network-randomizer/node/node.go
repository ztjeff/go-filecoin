package node

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ipfs/go-ipfs-files"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"

	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	lpfc "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/constants"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/plumbing"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/repo"
	"github.com/filecoin-project/go-filecoin/types"
)

// Node represents a full fcnr node.
type Node struct {
	host host.Host

	// Repo is the repo this node was created with
	// it contains all persistent artifacts of the filecoin node
	Repo repo.Repo

	PlumbingAPI *plumbing.API
}

// Config is a helper to aid in the construction of a filecoin node.
type Config struct {
	Libp2pOpts []libp2p.Option
	Repo       repo.Repo
}

// ConfigOpt is a configuration option for a filecoin node.
type ConfigOpt func(*Config) error

// Libp2pOptions returns a node config option that sets up the libp2p node
func Libp2pOptions(opts ...libp2p.Option) ConfigOpt {
	return func(nc *Config) error {
		// Quietly having your options overridden leads to hair loss
		if len(nc.Libp2pOpts) > 0 {
			panic("Libp2pOptions can only be called once")
		}
		nc.Libp2pOpts = opts
		return nil
	}
}

// New creates a new node.
func New(ctx context.Context, opts ...ConfigOpt) (*Node, error) {
	n := &Config{}
	for _, o := range opts {
		if err := o(n); err != nil {
			return nil, err
		}
	}

	return n.Build(ctx)
}

// Build instantiates a filecoin Node from the settings specified in the config.
func (nc *Config) Build(ctx context.Context) (*Node, error) {
	if nc.Repo == nil {
		panic("nil repo")
	}

	peerHost, err := libp2p.New(
		ctx,
		libp2p.ChainOptions(nc.Libp2pOpts...),
	)
	if err != nil {
		return nil, err
	}

	fastEnv, err := fast.NewEnvironmentMemoryGenesis(big.NewInt(2000000000), "~/.fcnr", types.TestProofsMode)
	if err != nil {
		return nil, err
	}

	if err := setupGenesisNode(ctx, fastEnv); err != nil {
		return nil, err
	}

	plumbingAPI := plumbing.New(&plumbing.APIDeps{
		FastEnv: fastEnv,
	})

	nd := &Node{
		host:        peerHost,
		Repo:        nc.Repo,
		PlumbingAPI: plumbingAPI,
	}

	return nd, nil
}

func setupGenesisNode(ctx context.Context, env fast.Environment) error {
	// Setup localfilecoin plugin options
	options := make(map[string]string)
	options[lpfc.AttrLogJSON] = "0"                      // Disable JSON logs
	options[lpfc.AttrLogLevel] = "4"                     // Set log level to Info
	options[lpfc.AttrFilecoinBinary] = constants.BinPath // Use the repo binary

	genesisURI := env.GenesisCar()
	genesisMiner, err := env.GenesisMiner()
	if err != nil {
		return err
	}

	fastenvOpts := fast.EnvironmentOpts{
		InitOpts:   []fast.ProcessInitOption{fast.POGenesisFile(genesisURI)},
		DaemonOpts: []fast.ProcessDaemonOption{fast.POBlockTime(constants.BlockTime)},
	}

	ctx = series.SetCtxSleepDelay(ctx, constants.BlockTime)

	// The genesis process is the filecoin node that loads the miner that is
	// define with power in the genesis block, and the prefunnded wallet
	genesis, err := env.NewProcess(ctx, lpfc.PluginName, options, fastenvOpts)
	if err != nil {
		return err
	}

	err = series.SetupGenesisNode(ctx, genesis, genesisMiner.Address, files.NewReaderFile(genesisMiner.Owner))
	if err != nil {
		return err
	}

	if err := genesis.MiningStart(ctx); err != nil {
		return err
	}
	return nil
}

// Start boots up the node.
func (node *Node) Start(ctx context.Context) error {
	fmt.Println("starting Filecoin Network Randomizer :)")
	fmt.Printf("Environment GenesisURL: %s\n", node.PlumbingAPI.FastEnvironment(ctx).GenesisCar())
	return nil
}

// Stop initiates the shutdown of the node.
func (node *Node) Stop(ctx context.Context) {

	if err := node.PlumbingAPI.FastEnvironment(ctx).Teardown(ctx); err != nil {
		fmt.Printf("error tearing down environment: %s\n", err)
	}

	if err := node.Host().Close(); err != nil {
		fmt.Printf("error closing host: %s\n", err)
	}

	if err := node.Repo.Close(); err != nil {
		fmt.Printf("error closing repo: %s\n", err)
	}

	fmt.Println("stopping Filecoin Network Randomizer :(")
}

// Host returns the nodes host.
func (node *Node) Host() host.Host {
	return node.host
}
