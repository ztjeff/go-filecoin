package randomizer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"

	logging "github.com/ipfs/go-log"
	lgwriter "github.com/ipfs/go-log/writer"
	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/actions"
	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/interfaces"
	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/network"
)

var log = logging.Logger("randomizer")

func init() {
	logging.SetAllLoggers(4)
	file, err := os.Create("./auditlogs.json")
	if err != nil {
		panic(err)
	}
	lgwriter.WriterGroup.AddWriter(file)

	dumbdumb()
}

func dumbdumb() {
	// I am aware this needs to be removed... but it works for now
	brute := exec.Command("bash", "-c", `cat setup.json | gengen --json > genesis.car 2> gen.json && cat gen.json | jq ".Miners[0].Address" > miner0 && cat gen.json | jq ".Miners[1].Address" > miner1 && cat gen.json | jq ".Miners[2].Address" > miner2 && cat gen.json | jq ".Keys[\"0\"]" > 0.key && cat gen.json | jq ".Keys[\"1\"]" > 1.key && cat gen.json | jq ".Keys[\"2\"]" > 2.key`)

	out, err := brute.Output()
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}

// TODO make gengen an importable package
func setupGenesis(count string) error {
	gensetup, err := exec.LookPath("setupgen")
	if err != nil {
		panic(err)
	}
	gengen, err := exec.LookPath("gengen")
	if err != nil {
		panic(err)
	}
	setup := exec.Command(gensetup, "-count", count)
	setupPipe, err := setup.StdoutPipe()
	if err != nil {
		panic(err)
	}
	genesis := exec.Command(gengen)
	if err != nil {
		panic(err)
	}
	genesis.Stdin = setupPipe

	if err := setup.Start(); err != nil {
		panic(err)
	}
	output, err := genesis.Output()
	if err != nil {
		panic(err)
	}
	if err := setup.Wait(); err != nil {
		panic(err)
	}
	genFile, err := os.Create("./genesis.car")
	if err != nil {
		panic(err)
	}
	defer genFile.Close()
	_, err = genFile.Write(output)
	if err != nil {
		panic(err)
	}
	return nil
	// jfc this gives me a panic attack...but it works :)
}

type BaseRandomizer struct {
	// Network is the network the randomizer operates over
	network randi.Network

	// Actions represent the set of actions the randomizer is capable of performing
	actions []randi.Action

	seed int64
}

func (b *BaseRandomizer) Network() (randi.Network, error) {
	return b.network, nil
}

func (b *BaseRandomizer) Actions() ([]randi.Action, error) {
	return b.actions, nil
}

func (b *BaseRandomizer) Attrs() (map[string]string, error) {
	panic("not implemented")
}

func (b *BaseRandomizer) Attr(key string) string {
	panic("not implemented")
}

func (b *BaseRandomizer) Events() (interface{}, error) {
	panic("not implemented")
}

func (b *BaseRandomizer) Seed() int64 {
	return b.seed
}

// TODO this is shoe-horned in here, not sure if Connect should be an
// action that a node performes, or somthing the network manages...
// Want this to have some preconditions -- thats the case for making it an action
// although actions are supposed to be random, and it doesn't make sense to have
// a random action that causes nodes to connect or disconnect.
func (bn *BaseRandomizer) Connect(ctx context.Context, n1, n2 testbedi.Core) (err error) {
	log.Infof("Randomizer connecting Node: %s to Node: %s", n1, n2)
	ctx = log.Start(ctx, "connect")
	defer func() {
		log.SetTags(ctx, map[string]interface{}{
			"node1": n1,
			"node2": n2,
		})
		log.FinishWithErr(ctx, err)
	}()
	return n1.Connect(ctx, n2)
}

func NewRandomizer() (randi.Randomizer, error) {
	randNet, err := network.NewNetwork("frristyNet", 3)
	if err != nil {
		return nil, err
	}

	// generate a seed
	seed := rand.Int63()

	var acts []randi.Action
	acts = append(acts, actions.NewInitAction())
	acts = append(acts, actions.NewDaemonAction())
	acts = append(acts, actions.NewWalletImportAction())
	acts = append(acts, actions.NewConfigAction())
	acts = append(acts, actions.NewMineOnceAction())

	baseRand := &BaseRandomizer{
		actions: acts,
		network: randNet,
		seed:    seed,
	}
	log.Infof("Create Ranzomizer with actions: %s, network: %s, seed: %d", baseRand.actions, baseRand.network.Name(), baseRand.Seed())
	return baseRand, nil
}
