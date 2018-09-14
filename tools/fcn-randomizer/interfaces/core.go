package randi

import (
	"context"

	tb "github.com/ipfs/iptb/testbed"
	"github.com/ipfs/iptb/testbed/interfaces"
)

// Network represents the set of nodes a randomizer controls
type Network interface {
	// A set of nodes and their configuration
	tb.Testbed
}

type Precondition interface {
	Name() string
	Condition(ctx context.Context, n testbedi.Core) (bool, error)
}

type Action interface {
	// Name is what the action is called
	Name() string

	// Runs a command in the context of the node
	Run(ctx context.Context, n testbedi.Core) (testbedi.Output, error)

	// Attributes are anything that shape the execution of an action, and example
	// could be: "maxBidPrice", this limits the price of a bid
	Attrs() map[string]string
	Attr(key string) string

	// Preconditions returns a slice of functions that must eval to true before
	// an action can be performed on a given node, an example could be:
	// checking the node has a miner address configured before trying to mine a block
	Preconditions() []Precondition
}

type Story interface {
	// Name of the story
	Name() string

	// Actions is a slice of all the actions making up a story
	Actions() []*Action

	// CurAction is the action that will be performed when the story is ran
	CurAction() *Action

	// A message with the result of the last action, not sure...
	Status() string
}

type Randomizer interface {
	// Network is the network the randomizer operates over
	Network() (Network, error)

	// Actions represent the set of actions the randomizer is capable of performing
	Actions() ([]Action, error)

	// Attributes are anything that shape the execution of an action, and example
	// could be: time between actions, limit on number of concurrent actions, etc.
	Attrs() (map[string]string, error)
	Attr(key string) string

	// Events represent an audit log of every action/story a randomizer has performed.
	// This will probs be a stream of ndjson, should also be written to a file
	Events() (interface{}, error)
}
