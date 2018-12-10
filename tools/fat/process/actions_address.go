package process

import (
	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	"github.com/filecoin-project/go-filecoin/address"
)

func (f *Filecoin) AddressNew() (address.Address, error) {
	var out address.Address

	if err := f.RunCmdJSON(&out, "go-filecoin", "address", "new"); err != nil {
		return address.Address{}, err
	}
	return out, nil
}
func (f *Filecoin) AddressLs() ([]string, error) {
	var out []string

	if err := f.RunCmdJSON(&out, "go-filecoin", "address", "ls"); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *Filecoin) AddressLookup(addr address.Address) (peer.ID, error) {
	var out peer.ID
	s_addr := addr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "address", "lookup", s_addr); err != nil {
		return "", err
	}
	return out, nil
}
