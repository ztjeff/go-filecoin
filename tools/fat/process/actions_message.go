package process

import (
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/commands"
	"github.com/filecoin-project/go-filecoin/types"
)

func (f *Filecoin) MessageWait(c cid.Cid) (commands.MessageWaitResult, error) {
	var out commands.MessageWaitResult

	if err := f.RunCmdJSON(&out, "go-filecoin", "message", "wait", c.String()); err != nil {
		return commands.MessageWaitResult{}, err
	}

	return out, nil
}

func (f *Filecoin) SendFilecoin(from, to address.Address, val *types.AttoFIL) (cid.Cid, error) {
	var out cid.Cid
	s_from := from.String()
	s_to := to.String()
	s_val := val.String()
	s_price := "0"
	s_limit := "0"

	if err := f.RunCmdJSON(&out, "go-filecoin", "message", "send", s_to, "--from", s_from, "--value", s_val, "--price", s_price, "--limit", s_limit); err != nil {
		return cid.Undef, err
	}
	return out, nil
}
