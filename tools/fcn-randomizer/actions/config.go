package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/actions/preconditions"
	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/interfaces"
)

type ConfigAction struct {
	name          string
	attributes    map[string]string
	preconditions []randi.Precondition
}

func (i *ConfigAction) Name() string {
	return i.name
}

func (i *ConfigAction) Run(ctx context.Context, n testbedi.Core, args ...string) (out testbedi.Output, err error) {
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

	for _, p := range i.Preconditions() {
		pass, err := p.Condition(ctx, n)
		if err != nil {
			return nil, err
		}
		if !pass {
			return nil, errors.New(fmt.Sprintf("precondition: %s failed", p.Name()))
		}
	}

	cmd := []string{"go-filecoin", "config"}
	cmd = append(cmd, args...)

	return n.RunCmd(ctx, nil, cmd...)
}

func (i *ConfigAction) Attrs() map[string]string {
	panic("not implemented")
}

func (i *ConfigAction) Attr(key string) string {
	panic("not implemented")
}

func (i *ConfigAction) Preconditions() []randi.Precondition {
	return i.preconditions
}

func NewConfigAction() randi.Action {
	var pc []randi.Precondition
	pc = append(pc, new(preconditions.HasRepo))
	return &ConfigAction{
		name:          "config",
		attributes:    nil,
		preconditions: pc,
	}
}
