package process

import (
	"strings"

	"gx/ipfs/QmZMWMvWMVKCbHetJ4RgndbuEF1io2UpUxwQwtNjtYPzSC/go-ipfs-files"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
)

func (f *Filecoin) WalletBalance(addr address.Address) (*types.AttoFIL, error) {
	// TODO will probably break with json
	var out types.AttoFIL
	s_addr := addr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "wallet", "balance", s_addr); err != nil {
		return nil, err
	}
	return &out, nil
}

func (f *Filecoin) WalletImport(file files.File) ([]address.Address, error) {
	var out []address.Address
	s_file := file.FullPath()

	if err := f.RunCmdJSON(&out, "go-filecoin", "wallet", "import", s_file); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *Filecoin) WalletExport(addrs []address.Address) ([]*types.KeyInfo, error) {
	var out []*types.KeyInfo
	var s_addrs []string
	for _, a := range addrs {
		s_addrs = append(s_addrs, a.String())
	}

	if err := f.RunCmdJSON(&out, "go-filecoin", "wallet", "export", strings.Join(s_addrs, " ")); err != nil {
		return nil, err
	}

	return out, nil
}
