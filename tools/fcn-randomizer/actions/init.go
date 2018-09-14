package actions

import (
	"context"
	"os"

	logging "github.com/ipfs/go-log"
	lgwriter "github.com/ipfs/go-log/writer"
	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/interfaces"
)

var log = logging.Logger("actions")

func init() {
	logging.SetAllLoggers(4)
	file, err := os.Create("./auditlogs.json")
	if err != nil {
		panic(err)
	}
	lgwriter.WriterGroup.AddWriter(file)
}

type InitAction struct {
	name          string
	attributes    map[string]string
	preconditions []randi.Precondition
}

func (i *InitAction) Name() string {
	return i.name
}

func (i *InitAction) Run(ctx context.Context, n testbedi.Core, args ...string) (out testbedi.Output, err error) {
	log.Infof("Node: %s Running go-filecoin %s %s", n, i.name, args)
	ctx = log.Start(ctx, i.name)
	defer func() {
		log.SetTags(ctx, map[string]interface{}{
			"node":     n,
			"cmd":      i.name,
			"args":     args,
			"exitcode": out.ExitCode(),
		})
		log.FinishWithErr(ctx, err)
	}()

	return n.Init(ctx, args...)
}

func (i *InitAction) Attrs() map[string]string {
	panic("not implemented")
}

func (i *InitAction) Attr(key string) string {
	panic("not implemented")
}

func (i *InitAction) Preconditions() []randi.Precondition {
	return i.preconditions
}

func NewInitAction() randi.Action {
	return &InitAction{
		name:          "init",
		attributes:    nil,
		preconditions: nil,
	}
}
