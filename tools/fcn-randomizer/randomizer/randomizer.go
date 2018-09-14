package randomizer

import (
	"context"
	"os"

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
}

type BaseRandomizer struct {
	// Network is the network the randomizer operates over
	network randi.Network

	// Actions represent the set of actions the randomizer is capable of performing
	actions []randi.Action
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

	initact := actions.NewInitAction()
	daemonact := actions.NewDaemonAction()

	var actions []randi.Action
	actions = append(actions, initact)
	actions = append(actions, daemonact)

	baseRand := &BaseRandomizer{
		actions: actions,
		network: randNet,
	}
	return baseRand, nil
}
