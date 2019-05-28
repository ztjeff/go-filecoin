package node

import (
	"context"
	"fmt"
	"math/big"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"

	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/repo"
	"github.com/filecoin-project/go-filecoin/types"
)

// Node represents a full fcnr node.
type Node struct {
	host host.Host

	// Repo is the repo this node was created with
	// it contains all persistent artifacts of the filecoin node
	Repo repo.Repo

	FastEnv fast.Environment
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

	nd := &Node{
		host:    peerHost,
		Repo:    nc.Repo,
		FastEnv: fastEnv,
	}

	return nd, nil
}

// Start boots up the node.
func (node *Node) Start(ctx context.Context) error {
	fmt.Println("starting Filecoin Network Randomizer :)")
	fmt.Printf("Environment GenesisURL: %s\n", node.FastEnv.GenesisCar())
	return nil
}

// Stop initiates the shutdown of the node.
func (node *Node) Stop(ctx context.Context) {

	if err := node.FastEnv.Teardown(ctx); err != nil {
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
