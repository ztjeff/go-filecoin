package randomizer

import (
	"github.com/filecoin-project/go-randomizer/actions"
	"github.com/filecoin-project/go-randomizer/interfaces"
	"github.com/filecoin-project/go-randomizer/network"
)

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
