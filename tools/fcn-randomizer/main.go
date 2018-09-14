package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	logging "github.com/ipfs/go-log"
	iptb "github.com/ipfs/iptb/testbed/interfaces"

	r "github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/randomizer"
)

var log = logging.Logger("main")

func main() {
	ctx := context.Background()
	randomizer, err := r.NewRandomizer()
	if err != nil {
		panic(err)
	}

	net, err := randomizer.Network()
	if err != nil {
		panic(err)
	}

	// get all the actions
	acts, err := randomizer.Actions()
	if err != nil {
		panic(err)
	}

	// get all the nodes in the network
	nodes, err := net.Nodes()
	if err != nil {
		panic(err)
	}

	// this is an mvp so the values are hardcoded
	initAct := acts[0]      // an action representing an init command
	daemAct := acts[1]      // an action representing a daemon command
	walletImport := acts[2] // an action representing a daemon command
	configAct := acts[3]
	mineOnceAct := acts[4]

	// lets itterate over all the nodes in the network and run actions against them
	// this is basically a story to init and start nodes
	for _, n := range nodes {
		// first we need to init the nodes repo and pass them a genesis file
		out, err := initAct.Run(ctx, n, "--genesisfile=./genesis.car")
		if err != nil {
			panic(err)
		}

		// check the exit code of the command
		if nonZeroExitCode(out, n) {
			break
		}

		// get the output of the command
		bytesOut, err := ioutil.ReadAll(out.Stdout())
		if err != nil {
			panic(err)
		}
		log.Infof("Node: %s Result: %s", n, string(bytesOut))

		// we done good so far, lets start the daemon
		out, err = daemAct.Run(ctx, n)
		if err != nil {
			panic(err)
		}
		log.Infof("Node: %s Result: daemon started ok!", n)
	}

	// the network will now connect the nodes
	if err := randomizer.Connect(ctx, nodes[0], nodes[1]); err != nil {
		panic(err)
	}
	if err := randomizer.Connect(ctx, nodes[1], nodes[2]); err != nil {
		panic(err)
	}

	for key, n := range nodes {
		// import a genesis key into our wallet
		out, err := walletImport.Run(ctx, n, fmt.Sprintf("./%d.key", key))
		if err != nil {
			panic(err)
		}

		if nonZeroExitCode(out, n) {
			break
		}

		bytesOut, err := ioutil.ReadAll(out.Stdout())
		if err != nil {
			panic(err)
		}
		log.Infof("Node: %s Result: %s", n, string(bytesOut))

		// now configure our miner adddress to work with said key
		// read the miner file TODO hack for now
		addr, err := ioutil.ReadFile(fmt.Sprintf("miner%d", key))
		if err != nil {
			panic(err)
		}

		// mining.minerAddresses "[\"fcqfg3ny24a8mzz4tjg2s2zuna5mah75vwrsssgzz\"]"
		out, err = configAct.Run(ctx, n, "mining.minerAddresses", fmt.Sprintf("[%s]", string(addr)))
		if err != nil {
			panic(err)
		}

		if nonZeroExitCode(out, n) {
			break
		}

		bytesOut, err = ioutil.ReadAll(out.Stdout())
		if err != nil {
			panic(err)
		}
		log.Infof("Node: %s Result: %s", n, string(bytesOut))
	}

	// now let make hella filecoin
	rand.Seed(randomizer.Seed())
	for {
		// pick a lucky winner!
		miner, err := net.Node(rand.Int() % 3)
		if err != nil {
			panic(err)
		}

		out, err := mineOnceAct.Run(ctx, miner)
		if err != nil {
			panic(err)
		}

		if nonZeroExitCode(out, miner) {
			break
		}

		bytesOut, err := ioutil.ReadAll(out.Stdout())
		if err != nil {
			panic(err)
		}
		log.Infof("Node: %s Result: %s", miner, string(bytesOut))
		time.Sleep(time.Second * 3)
	}

	log.Info("Complete")

	// this sleep allows defered event log calls to write TODO fix this
	time.Sleep(time.Second * 2)

}

func nonZeroExitCode(out iptb.Output, n iptb.Core) bool {
	// this is the case for starting the daemon
	if out == nil {
		return false
	}
	if out.ExitCode() != 0 {
		bytesErr, err := ioutil.ReadAll(out.Stderr())
		if err != nil {
			panic(err)
		}
		log.Errorf("Node: %s got non-zero exitcode: %d, stderr: %s", n, out.ExitCode(), string(bytesErr))
		return true
	}
	return false

}
