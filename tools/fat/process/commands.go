package process

import (
	"fmt"
	"io"
	"os"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"

	"github.com/filecoin-project/go-filecoin/address"
)

// Starts the daemon and cache its peerID
func (f *Filecoin) DaemonStart() error {
	log.Info("Starting Filecoin daemon")
	_, err := f.Start(f.ctx, false, "--block-time=5s")
	if err != nil {
		return err
	}

	pid, err := f.PeerID()
	if err != nil {
		return err
	}
	log.Infof("Started Daemon with peerID: %s", pid)

	f.ID, err = peer.IDB58Decode(pid)
	if err != nil {
		panic(err)
		return err
	}

	// start monitoring stderr
	if err := f.openLogWindow(); err != nil {
		return err
	}

	return nil
}

func (f *Filecoin) RetrievePiece(miner address.Address, data cid.Cid) (io.Reader, error) {
	s_data := data.String()
	s_miner := miner.String()

	out, err := f.RunCmd(f.ctx, nil, "go-filecoin", "retrieval-client", "retrieve-piece", s_miner, s_data)
	if err != nil {
		return nil, err
	}

	if out.ExitCode() > 0 {
		io.Copy(os.Stderr, out.Stderr())
		return nil, fmt.Errorf("Non zero exit code")
	}

	return out.Stdout(), nil
}

// TODO I think we may need to rething the types returned by these interface methods
// returning a channel or an interface is tough to deal with
/*
func (f *Filecoin) ListAsks(ctx context.Context) (<-chan Ask, error) {
}
func (f *Filecoin) Cat(c cid.Cid) (uio.DagReader, error) {
}
func (f *Filecoin) ImportData(data io.Reader) (ipld.Node, error) {
}

*/

// TODO not sure how this is going to work with bigints
/*
func (f *Filecoin) GetPledge(minerAddr address.Address) (*big.Int, error) {
	// TODO I don't think this is going to play nicely with json, just a gut feeling
	var out string
	s_minerAddr := minerAddr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "pledge", s_minerAddr); err != nil {
		return nil, err
	}

	return out, nil
}

func (f *Filecoin) GetPower(minerAddr address.Address) (*big.Int, error) {
	// TODO I don't think this is going to play nicely with json, just a gut feeling
	var put string
	s_minerAddr := minerAddr.String()

	if err := f.RunCmdJSON(&out, "go-filecoin", "miner", "power", s_minerAddr); err != nil {
		return nil, err
	}

	return out, nil
}
*/
