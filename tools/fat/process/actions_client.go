package process

import (
	"encoding/json"
	"fmt"
	"io"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/commands"
	"github.com/filecoin-project/go-filecoin/protocol/storage"
)

type AsksIterator interface {
	Next() (*commands.ClientListAsksResult, error)
	HasNext() bool
	Close() error
}

type asksIterator struct {
	ln LineIterator
}

func (i *asksIterator) Next() (*commands.ClientListAsksResult, error) {
	v := new(commands.ClientListAsksResult)
	bs, err := i.ln.Next()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bs, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (i *asksIterator) HasNext() bool {
	return i.ln.HasNext()
}

func (i *asksIterator) Close() error {
	return nil
}

func (f *Filecoin) ClientListAsks() (AsksIterator, error) {
	iter, err := f.RunCmdLDJSON("go-filecoin", "client", "list-asks")
	if err != nil {
		return nil, err
	}

	return &asksIterator{
		ln: iter,
	}, nil
}

func (f *Filecoin) ProposeStorageDeal(data cid.Cid, miner address.Address, ask uint64, duration uint64) (*storage.DealResponse, error) {
	var out storage.DealResponse
	s_data := data.String()
	s_miner := miner.String()
	s_ask := fmt.Sprintf("%d", ask)
	s_duration := fmt.Sprintf("%d", duration)

	if err := f.RunCmdJSON(&out, "go-filecoin", "client", "propose-storage-deal", s_miner, s_data, s_ask, s_duration); err != nil {
		return nil, err
	}
	return &out, nil
}

func (f *Filecoin) ClientImport(file io.Reader) (cid.Cid, error) {
	var out cid.Cid
	if err := f.RunCmdJSONWithStdin(&out, file, "go-filecoin", "client", "import"); err != nil {
		return cid.Undef, err
	}
	return out, nil
}

func (f *Filecoin) QueryStorageDeal(prop cid.Cid) (*storage.DealResponse, error) {
	var out storage.DealResponse
	s_prop := prop.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "client", "query-storage-deal", s_prop); err != nil {
		return nil, err
	}
	return &out, nil
}
