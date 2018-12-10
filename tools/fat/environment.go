package fat

import (
	"context"
	"fmt"
	"os"

	iptb "github.com/ipfs/iptb/testbed"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"

	"github.com/filecoin-project/go-filecoin/tools/fat/process"
)

// Environment is a structure which contains a set of filecoin processes
// and globally shared resources
type Environment struct {
	Location     string // ideally something more general than a path
	GenesisFile  *GenesisInfo
	GenesisMiner *process.Filecoin

	processes []*process.Filecoin
}

func LoadEnvironment(dir string) (*Environment, error) {
	return nil, nil
}

func SaveEnvironment(*Environment) error {
	return nil
}

func NewEnvironment(dir string) (*Environment, error) {
	log.Infof("Creating Environment: %s", dir)
	gf, err := GenerateGenesis(10000000, dir)
	if err != nil {
		return nil, err
	}

	return &Environment{
		Location:    dir,
		GenesisFile: gf,
		processes:   nil,
	}, nil
}

func (e *Environment) Processes() []*process.Filecoin {
	return e.processes
}

func (e *Environment) Teardown() error {
	for _, n := range e.processes {
		n.Stop(context.Background())
	}

	return os.RemoveAll(e.Location)
}

func (e *Environment) AddProcess(p *process.Filecoin) {
	for _, n := range e.processes {
		n.Connect(context.Background(), p)
	}
	e.processes = append(e.processes, p)
}

// TODO don't put "go-filecoin" in the path, tell the process what to use here
func (e *Environment) NewProcess(ctx context.Context, t, d string, attrs map[string]string) (*process.Filecoin, error) {
	log.Infof("NewProcess, type: %s, dir: %s", t, d)
	ns := iptb.NodeSpec{
		Type:  t,
		Dir:   d,
		Attrs: attrs,
	}

	if err := os.MkdirAll(d, 0775); err != nil {
		return nil, err
	}

	c, err := ns.Load()
	if err != nil {
		return nil, err
	}

	return process.NewProcess(ctx, t, d, c), nil
}

func (e *Environment) CreateGenesisMiner(ctx context.Context) (*process.Filecoin, error) {
	log.Info("Run CreateGenesisMiner")
	// Create a filecoin process structure
	fc, err := e.NewProcess(ctx, "localfilecoin", fmt.Sprintf("%s/0", e.Location), nil)
	if err != nil {
		return nil, err
	}

	// init the filecoin node with a genesis file
	_, err = fc.Init(ctx, "--genesisfile", e.GenesisFile.Path)
	if err != nil {
		return nil, err
	}

	// start the filecoin node
	err = fc.DaemonStart()
	if err != nil {
		return nil, err
	}

	// add the miner address to the nodes config
	_, err = fc.RunCmd(ctx, nil, "go-filecoin", "config", "mining.minerAddress", e.GenesisFile.MinerAddress.String())
	if err != nil {
		return nil, err
	}

	// import the miner address key into the nodes wallet
	_, err = fc.RunCmd(ctx, nil, "go-filecoin", "wallet", "import", e.GenesisFile.KeyFile)
	if err != nil {
		return nil, err
	}

	// update the nodes peerID to own said miner
	_, err = fc.UpdatePeerID(e.GenesisFile.WalletAddress, e.GenesisFile.MinerAddress, fc.ID)
	if err != nil {
		return nil, err
	}

	fc.DefaultWalletAddr = e.GenesisFile.WalletAddress
	fc.MinerOwner = e.GenesisFile.WalletAddress
	fc.MinerAddress = e.GenesisFile.MinerAddress

	if err := fc.MiningStart(); err != nil {
		return nil, err
	}

	return fc, nil
}

func (e *Environment) AddGenesisMiner(ctx context.Context) error {
	fc, err := e.CreateGenesisMiner(ctx)
	if err != nil {
		return err
	}
	e.GenesisMiner = fc
	e.AddProcess(fc)
	return nil
}

func (e *Environment) CreateNode(ctx context.Context) (*process.Filecoin, error) {
	// Create a filecoin process structure
	fc, err := e.NewProcess(ctx, "localfilecoin", fmt.Sprintf("%s/%d", e.Location, len(e.processes)), nil)
	if err != nil {
		return nil, err
	}

	// init the filecoin node with a genesis file
	_, err = fc.Init(ctx, "--genesisfile", e.GenesisFile.Path, "--auto-seal-interval-seconds=0")
	if err != nil {
		return nil, err
	}

	// start the filecoin node
	err = fc.DaemonStart()
	if err != nil {
		return nil, err
	}

	addrs, err := fc.AddressLs()
	if err != nil {
		return nil, err
	}

	// sanity check
	if len(addrs) != 1 {
		panic(addrs)
	}
	fc.DefaultWalletAddr, err = address.NewFromString(addrs[0])
	if err != nil {
		return nil, err
	}

	return fc, nil
}

func (e *Environment) AddNode(ctx context.Context) error {
	log.Info("Adding Node")
	fc, err := e.CreateNode(ctx)
	if err != nil {
		return err
	}
	e.AddProcess(fc)
	return nil
}

func (e *Environment) CreateMiner(ctx context.Context) (*process.Filecoin, error) {
	fc, err := e.CreateNode(ctx)
	if err != nil {
		return nil, err
	}
	e.AddProcess(fc)

	value, ok := types.NewAttoFILFromFILString("100")
	if !ok {
		panic("Was unable to make Atto from string")
	}

	c, err := e.GenesisMiner.SendFilecoin(e.GenesisMiner.DefaultWalletAddr, fc.DefaultWalletAddr, value)
	if err != nil {
		return nil, err
	}

	_, err = e.GenesisMiner.MiningOnce()
	if err != nil {
		return nil, err
	}

	fc.MessageWait(c)

	// TODO should probabaly wait on message and check receipt
	fc.MinerAddress, err = fc.CreateMiner(fc.DefaultWalletAddr, 100, fc.ID, value)
	if err != nil {
		return nil, err
	}
	fc.MinerOwner = fc.DefaultWalletAddr

	return fc, nil
}

func (e *Environment) AddMiner(ctx context.Context) error {
	_, err := e.CreateMiner(ctx)
	if err != nil {
		return err
	}
	return nil
}
