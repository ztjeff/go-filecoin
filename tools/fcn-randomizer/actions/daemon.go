package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/actions/preconditions"
	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/interfaces"
)

type DaemonAction struct {
	name          string
	attributes    map[string]string
	preconditions []randi.Precondition
}

func (i *DaemonAction) Name() string {
	return i.name
}

func (i *DaemonAction) Run(ctx context.Context, n testbedi.Core) (out testbedi.Output, err error) {
	log.Infof("Node: %s Running go-filecoin %s", n, i.name)
	ctx = log.Start(ctx, i.name)
	defer func() {
		log.SetTags(ctx, map[string]interface{}{
			"node":     n,
			"run":      i.name,
			"exitcode": out.ExitCode(),
		})
		log.FinishWithErr(ctx, err)
	}()

	for _, p := range i.Preconditions() {
		pass, err := p.Condition(ctx, n)
		if err != nil {
			return nil, err
		}
		if !pass {
			return nil, errors.New(fmt.Sprintf("precondition: %s failed", p.Name()))
		}
	}

	return n.Start(ctx, true)
}

func (i *DaemonAction) Attrs() map[string]string {
	panic("not implemented")
}

func (i *DaemonAction) Attr(key string) string {
	panic("not implemented")
}

func (i *DaemonAction) Preconditions() []randi.Precondition {
	return i.preconditions
}

func NewDaemonAction() randi.Action {
	var pc []randi.Precondition

	hasRepo := new(preconditions.HasRepo)

	pc = append(pc, hasRepo)
	return &DaemonAction{
		name:          "daemon",
		attributes:    nil,
		preconditions: pc,
	}
}
