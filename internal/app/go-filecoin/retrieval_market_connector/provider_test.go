package retrievalmarketconnector_test

import (
	"context"
	"math/rand"
	"testing"

	gfmtut "github.com/filecoin-project/go-fil-markets/shared_testutil"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"
	"github.com/ipfs/go-datastore"
	dss "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"

	. "github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/retrieval_market_connector"
)

func TestNewRetrievalProviderNodeConnector(t *testing.T) {

}

func TestRetrievalProviderConnector_UnsealSector(t *testing.T) {

}

func TestRetrievalProviderConnector_SavePaymentVoucher(t *testing.T) {
	rmnet := gfmtut.NewTestRetrievalMarketNetwork(gfmtut.TestNetworkParams{})
	ps := gfmtut.NewTestPieceStore()
	bs := blockstore.NewBlockstore(dss.MutexWrap(datastore.NewMapDatastore()))
	pchan, err := address.NewIDAddress(rand.Uint64())
	require.NoError(t, err)
	ctx := context.Background()
	voucher := &paych.SignedVoucher{
		Lane:            rand.Uint64(),
		Nonce:           rand.Uint64(),
		Amount:          big.NewInt(rand.Int63()),
		MinSettleHeight: abi.ChainEpoch(99),
	}
	dealAmount := big.NewInt(rand.Int63())
	proof := []byte("proof")

	t.Run("saves payment voucher and returns voucher amount if new", func(t *testing.T) {
		rpc := NewRetrievalProviderConnector(rmnet, ps, bs)
		tokenamt, err := rpc.SavePaymentVoucher(ctx, pchan, voucher, proof, dealAmount)
		assert.NoError(t, err)
		assert.True(t, voucher.Amount.Equals(tokenamt))
	})

	t.Run("errors and does not overwrite voucher if already saved, regardless of other params", func(t *testing.T) {
		rpc := NewRetrievalProviderConnector(rmnet, ps, bs)
		_, err := rpc.SavePaymentVoucher(ctx, pchan, voucher, proof, dealAmount)
		require.NoError(t, err)
		proof2 := []byte("newproof")
		dealAmount2 := big.NewInt(rand.Int63())
		pchan2, err := address.NewIDAddress(rand.Uint64())
		require.NoError(t, err)
		tokenamt2, err := rpc.SavePaymentVoucher(ctx, pchan2, voucher, proof2, dealAmount2)
		assert.EqualError(t, err, "voucher exists")
		assert.Equal(t, abi.NewTokenAmount(0), tokenamt2)
	})
	t.Run("errors if cannot create a key for the voucher store", func(t *testing.T) {
		rpc := NewRetrievalProviderConnector(rmnet, ps, bs)
		badVoucher := &paych.SignedVoucher{
			Merges: make([]paych.Merge, cbg.MaxLength+1),
		}

		_, err := rpc.SavePaymentVoucher(ctx, pchan, badVoucher, proof, dealAmount)
		assert.EqualError(t, err, "Slice value in field t.Merges was too long")
	})
}
