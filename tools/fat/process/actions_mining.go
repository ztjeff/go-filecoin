package process

import (
	"github.com/filecoin-project/go-filecoin/types"
)

func (f *Filecoin) MiningOnce() (*types.Block, error) {
	var out types.Block

	if err := f.RunCmdJSON(&out, "go-filecoin", "mining", "once"); err != nil {
		return nil, err
	}
	return &out, nil
}

// TODO check exit code
func (f *Filecoin) MiningStart() error {
	_, err := f.RunCmd(f.ctx, nil, "go-filecoin", "mining", "start")
	if err != nil {
		return err
	}
	return nil
}

// TODO check exit code
func (f *Filecoin) MiningStop() error {
	_, err := f.RunCmd(f.ctx, nil, "go-filecoin", "mining", "stop")
	if err != nil {
		return err
	}
	return nil
}
