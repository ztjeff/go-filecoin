package actions

import (
	"context"
	"os"

	logger "github.com/ipfs/go-log"
	lgwriter "github.com/ipfs/go-log/writer"
	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-randomizer/interfaces"
)

var log = logger.Logger("actions")

func init() {
	logger.SetAllLoggers(4)
	file, err := os.Create("/tmp/networkRandomizer_auditlogs.json")
	if err != nil {
		panic(err)
	}
	lgwriter.WriterGroup.AddWriter(file)
}

type InitAction struct {
	name          string
	attributes    map[string]string
	preconditions []func(n testbedi.Core) (bool, error)
}

func (i *InitAction) Name() string {
	return i.name
}

func (i *InitAction) Run(n testbedi.Core) (testbedi.Output, error) {
	log.Infof("Node: %s Running go-filecoin init", n)

	ctx := context.Background()

	ctx = log.Start(ctx, i.name)
	log.SetTag(ctx, "node", n)
	defer log.Finish(ctx)

	return n.Init(ctx)
}

func (i *InitAction) Attrs() map[string]string {
	panic("not implemented")
}

func (i *InitAction) Attr(key string) string {
	panic("not implemented")
}

func (i *InitAction) Preconditions() []func(n testbedi.Core) (bool, error) {
	return nil
}

func NewInitAction() randi.Action {
	return &InitAction{
		name:          "init",
		attributes:    nil,
		preconditions: nil,
	}
}
