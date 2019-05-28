package network

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gx/ipfs/QmXWZCd8jfaHmt4UDSnjKmGcrQMw95bDGWqEeVLVJjoANX/go-ipfs-files"

	"github.com/Pallinder/sillyname-go"
	"github.com/filecoin-project/go-filecoin/address"
	fast "github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	lpfc "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

type Ability int

const (
	Client Ability = iota
	Miner
)

var NetworkOptions map[string]string
var BinPath string

func init() {

	// Setup localfilecoin plugin options
	NetworkOptions := make(map[string]string)
	NetworkOptions[lpfc.AttrLogJSON] = "1"            // Enable JSON logs
	NetworkOptions[lpfc.AttrLogLevel] = "5"           // Set log level to Debug
	NetworkOptions[lpfc.AttrUseSmallSectors] = "true" // Enable small sectors
	NetworkOptions[lpfc.AttrFilecoinBinary] = BinPath // Use the repo binary}
}

var log = logging.Logger("network")

type Network struct {
	// means we are not connecting to a devnet
	genesisURL  string
	faucetURL   string
	localDeploy bool
	lk          sync.RWMutex
	nodes       []*FCNRNode
	repoNum     int
	repoDir     string
	env         fast.Environment
}

type FCNRNode struct {
	abilityMu sync.Mutex
	ability   Ability
	fastNode  *fast.Filecoin
}

func (rn *FCNRNode) TapFaucet(ctx context.Context, faucetURL string) error {
	fmt.Println("TapFaucet")
	var addr address.Address
	if err := rn.fastNode.ConfigGet(ctx, "wallet.defaultAddress", &addr); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s?target=%s", faucetURL, addr.String()), nil)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("TapFaucet Failed:", err)
		return err
	}
	fmt.Println("faucet response", resp.Status)
	return resp.Body.Close()
}

func (rn *FCNRNode) WaitForChainSync(ctx context.Context) error {
	var curHeight uint64
	curHeight = 0
	for {
		cids, err := rn.fastNode.ChainHead(ctx)
		if err != nil {
			rn.fastNode.DumpLastOutput(os.Stderr)
			return err
		}

		if len(cids) > 1 {
			fmt.Println("Chain has more than 1 head, syncing on", cids[0].String())
		}

		if curHeight > 0 {
			break
		}

		blk, err := rn.fastNode.ShowBlock(ctx, cids[0])
		if err != nil {
			rn.fastNode.DumpLastOutput(os.Stderr)
			return err
		}
		curHeight = uint64(blk.Height)
		fmt.Println("current head", blk.Cid())
		fmt.Println("current height", curHeight)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("Chain Sync Complete")
	return nil

}

func NewNetwork(repoDir string, genesisURL string, faucetURL string) (*Network, error) {
	env, err := fast.NewEnvironmentMemoryGenesis(big.NewInt(2000000000), repoDir)
	if err != nil {
		return nil, err
	}

	return &Network{repoDir: repoDir, env: env, genesisURL: genesisURL, faucetURL: faucetURL}, nil
}

func (n *Network) NewGenesisNode(ctx context.Context) (*fast.Filecoin, error) {
	// The genesis process is the filecoin node that loads the miner that is
	// define with power in the genesis block, and the prefunnded wallet
	genesis, err := n.env.NewProcess(ctx, lpfc.PluginName, NetworkOptions, fast.EnvironmentOpts{})
	if err != nil {
		return nil, err
	}

	genesisURI := n.env.GenesisCar()
	genesisMiner, err := n.env.GenesisMiner()
	if err != nil {
		return nil, err
	}

	if err := series.SetupGenesisNode(ctx, genesis, genesisURI, genesisMiner.Address, files.NewReaderFile(genesisMiner.Owner)); err != nil {
		return nil, err
	}

	return genesis, nil
}

func (n *Network) AddClient(ctx context.Context) (*fast.Filecoin, error) {
	fmt.Println("AddClient")
	client, err := n.env.NewProcess(ctx, lpfc.PluginName, NetworkOptions, fast.EnvironmentOpts{})
	if err != nil {
		return nil, err
	}

	// Start Client
	_, err = client.InitDaemon(ctx, "--devnet-user", "--genesisfile", n.genesisURL)
	if err != nil {
		client.DumpLastOutput(os.Stderr)
		return nil, err
	}
	fmt.Println("InitDaemon")

	_, err = client.StartDaemon(ctx, true)
	if err != nil {
		client.DumpLastOutput(os.Stderr)
		return nil, err
	}
	fmt.Println("StartDaemon")

	// connect to the dasboard
	if err := client.ConfigSet(ctx, "heartbeat.beatTarget", "/dns4/stats-infra.kittyhawk.wtf/tcp/8080/ipfs/QmUWmZnpZb6xFryNDeNU7KcJ1Af5oHy7fB9npU67sseEjR"); err != nil {
		client.DumpLastOutput(os.Stderr)
		return nil, err
	}

	// make a silly name, and remove spaces
	nicname := strings.Replace(sillyname.GenerateStupidName(), " ", "", -1)
	if err := client.ConfigSet(ctx, "heartbeat.nickname", nicname); err != nil {
		client.DumpLastOutput(os.Stderr)
		return nil, err
	}

	newNode := &FCNRNode{
		ability:  Client,
		fastNode: client,
	}

	// get the client funds from the faucet
	if err := newNode.TapFaucet(ctx, n.faucetURL); err != nil {
		client.StopDaemon(ctx)
		return nil, err
	}
	fmt.Println("Tap success")

	fmt.Println("Wait for chain sync")
	if err := newNode.WaitForChainSync(ctx); err != nil {
		return nil, err
	}

	// add it to the networks set of nodes once its chain is in sync
	n.lk.Lock()
	n.nodes = append(n.nodes, newNode)
	n.lk.Unlock()

	return client, nil
}

func (n *Network) AddMiner(ctx context.Context) (*fast.Filecoin, address.Address, error) {
	fmt.Println("addminer")
	miner, err := n.AddClient(ctx)
	if err != nil {
		return nil, address.Address{}, err
	}

	var addr address.Address
	if err := miner.ConfigGet(ctx, "wallet.defaultAddress", &addr); err != nil {
		panic(err)
		return nil, address.Address{}, err
	}

	pledge := uint64(10)          // sectors
	collateral := big.NewInt(100) // FIL

	fmt.Println("Trying to create Miner")
	minerAddr, err := miner.MinerCreate(ctx, pledge, collateral, fast.AOFromAddr(addr), fast.AOLimit(300), fast.AOPrice(big.NewFloat(1)))
	if err != nil {
		miner.DumpLastOutput(os.Stderr)
		return nil, address.Address{}, err
	}
	fmt.Println("miner created", minerAddr.String())
	fmt.Println("starting mining process...")

	if err := miner.MiningStart(ctx); err != nil {
		return nil, address.Address{}, err
	}

	fmt.Println("Mining started! miner creation complete!")

	return miner, minerAddr, nil
}

func (n *Network) ShutdownAll() error {
	n.lk.Lock()
	defer n.lk.Unlock()

	for _, fcn := range n.nodes {
		fcn.fastNode.StopDaemon(context.TODO())
	}
	return nil
}
