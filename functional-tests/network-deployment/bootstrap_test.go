package networkdeployment_test

import (
	"context"
	"testing"
	"time"

	//logging "github.com/ipfs/go-log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tf "github.com/filecoin-project/go-filecoin/testhelpers/testflags"
	"github.com/filecoin-project/go-filecoin/tools/fast/environment"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"

	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
)

func init() {
	//logging.SetDebugLogging()
}

// TestBootstrap verifies information about the bootstrap peers
func TestBootstrap(t *testing.T) {
	network := tf.DeploymentTest(t)
	ctx, env, f := setup(t, network)

	testBootstrap(ctx, t, env, f)
}

func testBootstrap(ctx context.Context, t *testing.T, env environment.Environment, foo *Foo) {
	client, err := env.NewProcess(ctx, localplugin.PluginName, foo.PluginOptions, foo.FastOptions)
	require.NoError(t, err)
	defer env.TeardownProcess(ctx, client)

	err = series.InitAndStart(ctx, client, foo.ConfigFn)
	require.NoError(t, err)

	t.Run("Check that we are connected to bootstrap peers", func(t *testing.T) {
		maddrChan := make(chan multiaddr.Multiaddr, 16)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			defer close(maddrChan)
			protop2p := multiaddr.ProtocolWithCode(multiaddr.P_P2P)
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
					peers, err := client.SwarmPeers(ctx)
					assert.NoError(t, err)

					for _, peer := range peers {
						transport, err := multiaddr.NewMultiaddr(peer.Addr)
						require.NoError(t, err)

						// /ipfs/<ID>
						peercomp, err := multiaddr.NewComponent(protop2p.Name, peer.Peer)
						require.NoError(t, err)

						fullmaddr := transport.Encapsulate(peercomp)
						maddrChan <- fullmaddr
					}
				}
			}
		}()

		bootstrapAddrs := networkBootstrapPeers(foo.Network)
		require.NotEmpty(t, bootstrapAddrs)

		bootstrapPeers, err := createResolvedPeerInfoMap(ctx, bootstrapAddrs)
		require.NoError(t, err)

		for maddr := range maddrChan {
			pinfo, err := pstore.InfoFromP2pAddr(maddr)
			require.NoError(t, err)

			if _, ok := bootstrapPeers[pinfo.ID]; !ok {
				continue
			}

			// pinfo will have only a single address as it comes from a single multiaddr
			require.NotEmpty(t, pinfo.Addrs)
			addr := pinfo.Addrs[0]

			t.Logf("Looking at addr %s", addr)
			for _, a := range bootstrapPeers[pinfo.ID].Addrs {
				if addr.Equal(a) {
					t.Logf("Found addr for peer %s", pinfo.ID)
					delete(bootstrapPeers, pinfo.ID)
				}
			}

			if len(bootstrapPeers) == 0 {
				cancel()
			}

			for peerID := range bootstrapPeers {
				t.Logf("Still waiting for %s", peerID)
			}
		}
	})
}
