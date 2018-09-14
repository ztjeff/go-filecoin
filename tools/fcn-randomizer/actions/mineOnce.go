package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/iptb/testbed/interfaces"

	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/actions/preconditions"
	"github.com/filecoin-project/go-filecoin/tools/fcn-randomizer/interfaces"
)

type MineOnceAction struct {
	name          string
	attributes    map[string]string
	preconditions []randi.Precondition
}

func (i *MineOnceAction) Name() string {
	return i.name
}

func (i *MineOnceAction) Run(ctx context.Context, n testbedi.Core, args ...string) (out testbedi.Output, err error) {
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

	cmd := []string{"go-filecoin", "mining", "once"}
	cmd = append(cmd, args...)

	return n.RunCmd(ctx, nil, cmd...)
}

func (i *MineOnceAction) Attrs() map[string]string {
	panic("not implemented")
}

func (i *MineOnceAction) Attr(key string) string {
	panic("not implemented")
}

func (i *MineOnceAction) Preconditions() []randi.Precondition {
	return i.preconditions
}

func NewMineOnceAction() randi.Action {
	var pc []randi.Precondition

	hasRepo := new(preconditions.HasRepo)

	pc = append(pc, hasRepo)
	return &MineOnceAction{
		name:          "mining once",
		attributes:    nil,
		preconditions: pc,
	}
}
