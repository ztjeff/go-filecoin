package network

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"math/big"
	"math/rand"
	"os"
	"time"

	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"

	address "github.com/filecoin-project/go-filecoin/address"
	//api "github.com/filecoin-project/go-filecoin/api"
	fast "github.com/filecoin-project/go-filecoin/tools/fast"
)

func init() {
	var ansiGray = "\033[0;37m"
	var ansiBlue = "\033[0;34m"
	logging.SetAllLoggers(4)
	logging.LogFormats["color"] = ansiGray + "%{time:15:04:05.000} %{color}%{level:5.5s} " + ansiBlue +
		"%{module}: %{color:reset}%{message} " + ansiGray + "%{shortfile}%{color:reset}"
}

type Action int

const (
	ActionPayment Action = iota
	ActionAsk
	ActionBid
	ActionDeal
	ActionSendFile
)

type Args struct {
	StartNodes     int
	MaxNodes       int
	JoinTime       time.Duration
	BlockTime      time.Duration
	ActionTime     time.Duration
	TestfilesDir   string
	GenesisFileURL string
	FaucetURL      string
	Actions        ActionArgs
}

type ActionArgs struct {
	Ask     bool
	Deal    bool
	Payment bool
	Mine    bool
}

type Randomizer struct {
	Net     *Network
	Args    Args
	Actions []Action
}

func periodic(ctx context.Context, t time.Duration, periodicFunc func(ctx context.Context)) {
	for {
		time.Sleep(t)

		select {
		case <-ctx.Done():
			return
		default:
		}

		periodicFunc(ctx)
	}
}

func NewRandomizer(n *Network, a Args) *Randomizer {
	r := &Randomizer{
		Net:     n,
		Args:    a,
		Actions: []Action{},
	}

	addif := func(t bool, a Action) {
		if t {
			r.Actions = append(r.Actions, a)
		}
	}
	addif(a.Actions.Ask, ActionAsk)
	addif(a.Actions.Deal, ActionDeal)
	addif(a.Actions.Payment, ActionPayment)

	return r
}

func (r *Randomizer) Run(ctx context.Context) error {
	fmt.Println("\nRandomizer running with params:")
	fmt.Println(StructToString(&r.Args))

	priceGen := NewPriceGenerator(1, 2, 1)

	var clients []*fast.Filecoin
	for i := 0; i < 1; i++ {
		c, err := r.Net.AddClient(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to add client")
		}
		fmt.Printf("client %d created\n", i)

		id, err := c.ID(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get client information")
		}

		for _, c := range clients {
			_, err := c.SwarmConnect(ctx, id.Addresses...)
			if err != nil {
				c.DumpLastOutput(os.Stderr)
				fmt.Println("CONNECT TO PEER FAILED: ", err)
			}
		}
		clients = append(clients, c)
	}

	var minerAddrs []address.Address
	for i := 0; i < 10; i++ {
		m, addr, err := r.SetupMiner(ctx, priceGen.GenerateNewPrice())
		if err != nil {
			return errors.Wrap(err, "failed to set up miner")
		}

		id, err := m.ID(ctx)
		if err != nil {
			m.DumpLastOutput(os.Stderr)
			return errors.Wrap(err, "failed to get miner ID")
		}

		for _, c := range clients {
			_, err := c.SwarmConnect(ctx, id.Addresses...)
			if err != nil {
				m.DumpLastOutput(os.Stderr)
				fmt.Println("CONNECT TO PEER FAILED: ", err)
			}
		}

		fmt.Printf("created miner %d: %s\n", i, addr)
		minerAddrs = append(minerAddrs, addr)
	}
	// no periodically add new nodes

	for {
		fmt.Println("Gonna store a file!")

		c := clients[rand.Intn(len(clients))]

		if err := ClientStoreFile(ctx, c, minerAddrs); err != nil {
			fmt.Println("Failed to store a file...", err)
		} else {
			fmt.Println("Stored a file!")
		}

		time.Sleep(time.Second * 10)
	}

	return nil
}

func (r *Randomizer) SetupMiner(ctx context.Context, price float64) (*fast.Filecoin, address.Address, error) {
	// add a miner
	miner, minerAddr, err := r.Net.AddMiner(ctx)
	if err != nil {
		return nil, address.Address{}, errors.Wrap(err, "failed to add miner")
	}
	miner.Log.Infof("Created Miner with price: %f", price)

	// set the price for the miner
	bPrice := big.NewFloat(price)           // price per byte/block
	expiry := big.NewInt(24 * 60 * 60 / 30) // ~24 hours
	pinfo, err := miner.MinerSetPrice(ctx, bPrice, expiry, fast.AOPrice(big.NewFloat(0.001)), fast.AOLimit(300))
	if err != nil {
		miner.DumpLastOutput(os.Stderr)
		return nil, address.Address{}, errors.Wrap(err, "miner set price failed")
	}
	fmt.Println("price set", pinfo)

	return miner, minerAddr, nil
}

func ClientStoreFile(ctx context.Context, c *fast.Filecoin, maddrs []address.Address) error {
	rf, err := RandomFileText()
	if err != nil {
		return errors.Wrap(err, "failed to generate random text file")
	}
	rfCID, err := c.ClientImport(ctx, rf)
	if err != nil {
		c.DumpLastOutput(os.Stderr)
		return errors.Wrap(err, "failed to import file on client")
	}
	c.Log.Infof("imported file cid: %s", rfCID.String())

	/*
			var asks []api.Ask
			dec, err := c.ClientListAsks(ctx)
			if err != nil {
				return errors.Wrap(err, "failed to list asks")
			}

			for {
				var a api.Ask
				if err := dec.Decode(&a); err != nil {
					fmt.Println("decode error:", err)
					break
				}
				asks = append(asks, a)
			}
		// pick a random miner
		a := asks[rand.Intn(len(asks))]
		maddr := a.Miner
	*/

	maddr := maddrs[rand.Intn(len(maddrs))]
	fmt.Println("attempting to store a file with: ", maddr)
	dealResponse, err := c.ClientProposeStorageDeal(ctx, rfCID, maddr, 0, 1000, false)
	if err != nil {
		c.DumpLastOutput(os.Stderr)
		return errors.Wrap(err, "client propose storage deal failed")
	}

	c.Log.Infof("deal response: %s", dealResponse)

	return nil
}
