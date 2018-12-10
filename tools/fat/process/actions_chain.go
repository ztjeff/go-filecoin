package process

import (
	"encoding/json"

	"github.com/filecoin-project/go-filecoin/commands"
)

type TipsetIterator interface {
	Next() (*commands.ChainLsResult, error)
	HasNext() bool
	Close() error
}

type tipsetIterator struct {
	ln LineIterator
}

func (i *tipsetIterator) Next() (*commands.ChainLsResult, error) {
	v := new(commands.ChainLsResult)
	bs, err := i.ln.Next()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bs, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (i *tipsetIterator) HasNext() bool {
	return i.ln.HasNext()
}

func (i *tipsetIterator) Close() error {
	return nil
}

func (f *Filecoin) ChainLs() (TipsetIterator, error) {
	iter, err := f.RunCmdLDJSON("go-filecoin", "chain", "ls")
	if err != nil {
		return nil, err
	}

	return &tipsetIterator{
		ln: iter,
	}, nil
}
