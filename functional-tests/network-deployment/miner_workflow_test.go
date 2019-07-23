package networkdeployment_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"math/big"
	"os"
	"testing"

	"github.com/ipfs/go-ipfs-files"
	logging "github.com/ipfs/go-log"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/protocol/storage/storagedeal"
	tf "github.com/filecoin-project/go-filecoin/testhelpers/testflags"
	"github.com/filecoin-project/go-filecoin/tools/fast"
	"github.com/filecoin-project/go-filecoin/tools/fast/environment"
	"github.com/filecoin-project/go-filecoin/tools/fast/series"
	localplugin "github.com/filecoin-project/go-filecoin/tools/iptb-plugins/filecoin/local"
	"github.com/filecoin-project/go-filecoin/types"
)

func init() {
	logging.SetDebugLogging()
}

func TestMiner(t *testing.T) {
	network := tf.DeploymentTest(t)
	ctx, env, f := setup(t, network)
	testMiner(ctx, t, env, f)
}

func testMiner(ctx context.Context, t *testing.T, env environment.Environment, foo *Foo) {
	miner, err := env.NewProcess(ctx, localplugin.PluginName, foo.PluginOptions, foo.FastOptions)
	require.NoError(t, err)
	defer env.TeardownProcess(ctx, miner)

	client, err := env.NewProcess(ctx, localplugin.PluginName, foo.PluginOptions, foo.FastOptions)
	require.NoError(t, err)
	defer env.TeardownProcess(ctx, client)

	// Start Miner
	err = series.InitAndStart(ctx, miner, foo.ConfigFn)
	require.NoError(t, err)

	// Start Client
	err = series.InitAndStart(ctx, client, foo.ConfigFn)
	require.NoError(t, err)

	// Everyone needs FIL to deal with gas costs and make sure their wallets
	// exists (sending FIL to a wallet addr creates it)
	err = env.GetFunds(ctx, miner)
	require.NoError(t, err)

	err = env.GetFunds(ctx, client)
	require.NoError(t, err)

	t.Run("Verify mining", func(t *testing.T) {
		collateral := big.NewInt(10)
		price := big.NewFloat(0.000000001)
		expiry := big.NewInt(128)

		defer client.DumpLastOutput(os.Stdout)
		defer miner.DumpLastOutput(os.Stdout)

		pparams, err := miner.Protocol(ctx)
		require.NoError(t, err)

		sectorSize := pparams.SupportedSectorSizes[0]

		// Create a miner on the miner node
		ask, err := series.CreateStorageMinerWithAsk(ctx, miner, collateral, price, expiry, sectorSize)
		require.NoError(t, err)

		// Connect the client and the miner
		err = series.Connect(ctx, client, miner)
		require.NoError(t, err)

		// Store some data with the miner with the given ask, returns the cid for
		// the imported data, and the deal which was created
		var data bytes.Buffer
		dataReader := io.LimitReader(rand.Reader, 512)
		dataReader = io.TeeReader(dataReader, &data)
		_, deal, err := series.ImportAndStoreWithDuration(ctx, client, ask, 32, files.NewReaderFile(dataReader))
		require.NoError(t, err)

		vouchers, err := client.ClientPayments(ctx, deal.ProposalCid)
		require.NoError(t, err)

		lastVoucher := vouchers[len(vouchers)-1]

		// Wait for the deal to be complete
		err = series.WaitForDealState(ctx, client, deal, storagedeal.Complete)
		require.NoError(t, err)

		// Redeem
		err = series.WaitForBlockHeight(ctx, miner, &lastVoucher.ValidAt)
		require.NoError(t, err)

		var addr address.Address
		err = miner.ConfigGet(ctx, "wallet.defaultAddress", &addr)
		require.NoError(t, err)

		balanceBefore, err := miner.WalletBalance(ctx, addr)
		require.NoError(t, err)

		mcid, err := miner.DealsRedeem(ctx, deal.ProposalCid, fast.AOPrice(big.NewFloat(1.0)), fast.AOLimit(300))
		require.NoError(t, err)

		result, err := miner.MessageWait(ctx, mcid)
		require.NoError(t, err)

		balanceAfter, err := miner.WalletBalance(ctx, addr)
		require.NoError(t, err)

		// We add the receipt back to the after balance to "undo" the gas costs, then substract the before balance
		// what is left is the change as a result of redeeming the voucher
		require.Equal(t, lastVoucher.Amount, balanceAfter.Add(result.Receipt.GasAttoFIL).Sub(balanceBefore))

		// Verify that the miner power has increased
		_, err = series.WaitForChainMessage(ctx, miner, func(ctx context.Context, node *fast.Filecoin, msg *types.SignedMessage) (bool, error) {
			if msg.Method == "submitPoSt" && msg.To == ask.Miner {
				return true, nil
			}

			return false, nil
		})
		require.NoError(t, err)

		mpower, err := miner.MinerPower(ctx, ask.Miner)
		require.NoError(t, err)

		// We should have a single sector of power
		require.Equal(t, &mpower.Power, sectorSize)

	})
}
