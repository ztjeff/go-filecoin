package actions

import (
	"context"
	"os"

	logger "github.com/ipfs/go-log"
	lgwriter "github.com/ipfs/go-log/writer"
	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-randomizer/interfaces"
)

func init() {
	logger.SetAllLoggers(4)
	file, err := os.Create("/tmp/networkRandomizer_auditlogs.json")
	if err != nil {
		panic(err)
	}
	lgwriter.WriterGroup.AddWriter(file)
}

type DaemonAction struct {
	name          string
	attributes    map[string]string
	preconditions []func(n testbedi.Core) (bool, error)
}

func (i *DaemonAction) Name() string {
	return i.name
}

func (i *DaemonAction) Run(n testbedi.Core) (testbedi.Output, error) {
	log.Infof("Node: %s Running go-filecoin daemon", n)

	ctx := context.Background()

	ctx = log.Start(ctx, i.name)
	log.SetTag(ctx, "node", n)
	defer log.Finish(ctx)

	return n.Start(ctx, true)
}

func (i *DaemonAction) Attrs() map[string]string {
	panic("not implemented")
}

func (i *DaemonAction) Attr(key string) string {
	panic("not implemented")
}

func (i *DaemonAction) Preconditions() []func(n testbedi.Core) (bool, error) {
	return nil
}

func NewDaemonAction() randi.Action {
	return &DaemonAction{
		name:          "daemon",
		attributes:    nil,
		preconditions: nil,
	}
}
