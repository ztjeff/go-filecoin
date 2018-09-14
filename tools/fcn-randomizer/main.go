package main

import (
	"context"
	"io/ioutil"
	"time"

	logging "github.com/ipfs/go-log"
	iptb "github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/randomizer"
)

var log = logging.Logger("main")

func main() {
	ctx := context.Background()
	rand, err := randomizer.NewRandomizer()
	if err != nil {
		panic(err)
	}

	net, err := rand.Network()
	if err != nil {
		panic(err)
	}

	// get all the actions
	acts, err := rand.Actions()
	if err != nil {
		panic(err)
	}

	// get all the nodes in the network
	nodes, err := net.Nodes()
	if err != nil {
		panic(err)
	}

	// this is an mvp so the values are hardcoded
	initAct := acts[0] // an action representing an init command
	daemAct := acts[1] // an action representing a daemon command

	// lets itterate over all the nodes in the network and run actions against them
	// this is basically a story to init and start nodes
	for _, n := range nodes {
		// first we need to init the nodes repo
		out, err := initAct.Run(ctx, n)
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
	if err := rand.Connect(ctx, nodes[0], nodes[1]); err != nil {
		panic(err)
	}
	if err := rand.Connect(ctx, nodes[1], nodes[2]); err != nil {
		panic(err)
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
