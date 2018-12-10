package process

import (
	"fmt"
	"math/big"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
)

func (f *Filecoin) CreateMiner(fromAddr address.Address, pledge uint64, pid peer.ID, collateral *types.AttoFIL) (address.Address, error) {
	var out address.Address
	s_fromAddr := fromAddr.String()
	s_pledge := fmt.Sprintf("%d", pledge)
	s_pid := pid.Pretty()
	s_collateral := collateral.String()
	s_price := "0"
	s_limit := "0"

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "create", s_pledge, s_collateral, "--from", s_fromAddr, "--peerid", s_pid, "--price", s_price, "--limit", s_limit); err != nil {
		return address.Address{}, err
	}

	return out, nil
}

func (f *Filecoin) UpdatePeerID(fromAddr, minerAddr address.Address, newPid peer.ID) (cid.Cid, error) {
	var out cid.Cid
	s_fromAddr := fromAddr.String()
	s_minerAddr := minerAddr.String()
	s_newPid := newPid.Pretty()
	s_price := "0"
	s_limit := "0"

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "update-peerid", s_minerAddr, s_newPid, "--from", s_fromAddr, "--price", s_price, "--limit", s_limit); err != nil {
		return cid.Undef, err
	}

	return out, nil
}

func (f *Filecoin) AddAsk(fromAddr, minerAddr address.Address, fil *types.AttoFIL, expiry *big.Int) (cid.Cid, error) {
	var out cid.Cid
	s_fromAddr := fromAddr.String()
	s_minerAddr := minerAddr.String()
	s_fil := fil.String()
	s_expiry := expiry.String()
	s_price := "0"
	s_limit := "0"

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "add-ask", s_minerAddr, s_fil, s_expiry, "--from", s_fromAddr, "--price", s_price, "--limit", s_limit); err != nil {
		return cid.Undef, err
	}
	return out, nil
}

func (f *Filecoin) GetOwner(minerAddr address.Address) (address.Address, error) {
	var out address.Address
	s_minerAdd := minerAddr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "owner", s_minerAdd); err != nil {
		return address.Address{}, err
	}

	return out, nil
}
