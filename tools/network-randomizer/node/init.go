package node

import (
	"context"

	ci "github.com/libp2p/go-libp2p-crypto"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/tools/network-randomizer/repo"
)

var ErrLittleBits = errors.New("Bitsize less than 1024 is considered unsafe") // nolint: golint

// InitCfg contains configuration for initializing a node
type InitCfg struct {
	PeerKey ci.PrivKey
}

// InitOpt is an init option function
type InitOpt func(*InitCfg)

// Init initializes a filecoin node in the given repo.
func Init(ctx context.Context, r repo.Repo, opts ...InitOpt) error {
	cfg := new(InitCfg)
	for _, o := range opts {
		o(cfg)
	}

	if cfg.PeerKey == nil {
		peerKey, err := makePrivateKey(2048)
		if err != nil {
			return errors.Wrap(err, "failed to create nodes private key")
		}
		cfg.PeerKey = peerKey
	}

	if err := r.Keystore().Put("self", cfg.PeerKey); err != nil {
		return errors.Wrap(err, "failed to store private key")
	}

	newConfig := r.Config()

	if err := r.ReplaceConfig(newConfig); err != nil {
		return errors.Wrap(err, "failed to update config with new values")
	}

	return nil
}

// makePrivateKey generates a new private key, which is the basis for a libp2p identity.
// borrowed from go-ipfs: `repo/config/init.go`
func makePrivateKey(nbits int) (ci.PrivKey, error) {
	if nbits < 1024 {
		return nil, ErrLittleBits
	}

	// create a public private key pair
	sk, _, err := ci.GenerateKeyPair(ci.RSA, nbits)
	if err != nil {
		return nil, err
	}

	return sk, nil
}
