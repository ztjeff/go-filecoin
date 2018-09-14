package network

import (
	"io/ioutil"

	iptb "github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"

	plugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
	"github.com/filecoin-project/go-randomizer/interfaces"
)

// this ensures the filecoin plugin has been loaded into iptb
func init() {
	_, err := iptb.RegisterPlugin(iptb.IptbPlugin{
		From:       "<builtin>",
		NewNode:    plugin.NewNode,
		PluginName: plugin.PluginName,
		BuiltIn:    true,
	}, false)

	if err != nil {
		panic(err)
	}
}

type BaseNetwork struct {
	// The name of the network
	name string

	// a network of iptb nodes
	network iptb.Testbed
}

func (bn *BaseNetwork) Name() string {
	return bn.name
}

func (bn *BaseNetwork) Node(n int) (testbedi.Core, error) {
	return bn.network.Node(n)
}

func (bn *BaseNetwork) Nodes() ([]testbedi.Core, error) {
	return bn.network.Nodes()
}

func (bn *BaseNetwork) Spec(n int) (*iptb.NodeSpec, error) {
	return bn.network.Spec(n)
}

func (bn *BaseNetwork) Specs() ([]*iptb.NodeSpec, error) {
	return bn.network.Specs()
}

func NewNetwork(name string, count int) (randi.Network, error) {
	networkDir, err := ioutil.TempDir("", name)
	if err != nil {
		return nil, err
	}

	testbed := iptb.NewTestbed(networkDir)

	specs, err := iptb.BuildSpecs(testbed.Dir(), count, plugin.PluginName, nil)
	if err != nil {
		return nil, err
	}

	if err := iptb.WriteNodeSpecs(testbed.Dir(), specs); err != nil {
		return nil, err
	}

	return &BaseNetwork{
		name:    name,
		network: &testbed,
	}, nil
}
