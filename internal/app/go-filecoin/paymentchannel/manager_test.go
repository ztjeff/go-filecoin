package paymentchannel_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/shared_testutil"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"
	"github.com/filecoin-project/specs-actors/actors/runtime/exitcode"
	spect "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dss "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/paymentchannel"
	"github.com/filecoin-project/go-filecoin/internal/app/go-filecoin/plumbing/cst"
	"github.com/filecoin-project/go-filecoin/internal/pkg/block"
	"github.com/filecoin-project/go-filecoin/internal/pkg/crypto"
)

func TestManager_GetPaymentChannelInfo(t *testing.T) {
	t.Run("returns err if info does not exist", func(t *testing.T) {
		ds := dss.MutexWrap(datastore.NewMapDatastore())
		ctx := context.Background()
		testAPI := NewFakePaymentChannelAPI(ctx, t)
		root := shared_testutil.GenerateCids(1)[0]
		viewer := makeStateViewer(t, root, nil)
		m := NewManager(context.Background(), ds, testAPI, testAPI, viewer, &cst.ChainStateReadWriter{})
		res, err := m.GetPaymentChannelInfo(spect.NewIDAddr(t, 1020))
		assert.EqualError(t, err, "No state for /t01020: datastore: key not found")
		assert.Nil(t, res)
	})
}

func TestManager_CreatePaymentChannel(t *testing.T) {
	ds := dss.MutexWrap(datastore.NewMapDatastore())
	ctx := context.Background()
	testAPI := NewFakePaymentChannelAPI(ctx, t)
	root := shared_testutil.GenerateCids(1)[0]
	viewer := makeStateViewer(t, root, nil)

	t.Run("happy path", func(t *testing.T) {
		m := NewManager(context.Background(), ds, testAPI, testAPI, viewer, &cst.ChainStateReadWriter{})
		clientAddr, minerAddr, paychIDAddr, paychUniqueAddr, _ := requireSetupPaymentChannel(t, testAPI, m)
		exists, err := m.ChannelExists(paychUniqueAddr)
		require.NoError(t, err)
		assert.True(t, exists)

		chinfo, err := m.GetPaymentChannelInfo(paychUniqueAddr)
		require.NoError(t, err)
		require.NotNil(t, chinfo)
		expectedChinfo := ChannelInfo{
			LastLane: 0,
			From:     clientAddr,
			To:       minerAddr,
			IDAddr:   paychIDAddr,
		}
		assert.Equal(t, expectedChinfo, *chinfo)
	})
	testCases := []struct {
		name             string
		waitErr, sendErr error
		expErr           string
	}{
		{name: "returns err and does not create channel if Send fails", sendErr: errors.New("sendboom"), expErr: "sendboom"},
		{name: "returns err and does not create channel if Wait fails", waitErr: errors.New("waitboom"), expErr: "waitboom"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testAPI.MsgSendErr = tc.sendErr
			testAPI.MsgWaitErr = tc.waitErr
			clientAddr := spect.NewIDAddr(t, 901)
			minerAddr := spect.NewIDAddr(t, 902)
			paychIDAddr := spect.NewIDAddr(t, 999)
			paychUniqueAddr := spect.NewActorAddr(t, "paych")
			blockHeight := uint64(1234)
			m := NewManager(context.Background(), ds, testAPI, testAPI, viewer, &cst.ChainStateReadWriter{})

			testAPI.StubCreatePaychActorMessage(clientAddr, minerAddr, paychIDAddr, paychUniqueAddr, builtin.MethodsInit.Exec, exitcode.Ok, blockHeight)

			err := m.CreatePaymentChannel(clientAddr, minerAddr)
			assert.EqualError(t, err, tc.expErr)
		})
	}
}

func TestManager_AllocateLane(t *testing.T) {
	ds := dss.MutexWrap(datastore.NewMapDatastore())
	ctx := context.Background()
	testAPI := NewFakePaymentChannelAPI(ctx, t)

	root := shared_testutil.GenerateCids(1)[0]
	viewer := makeStateViewer(t, root, nil)
	m := NewManager(context.Background(), ds, testAPI, testAPI, viewer, &cst.ChainStateReadWriter{})
	clientAddr, minerAddr, paychIDAddr, paychUniqueAddr, _ := requireSetupPaymentChannel(t, testAPI, m)

	t.Run("saves a new lane", func(t *testing.T) {
		lane, err := m.AllocateLane(paychUniqueAddr)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), lane)

		chinfo, err := m.GetPaymentChannelInfo(paychUniqueAddr)
		require.NoError(t, err)
		require.NotNil(t, chinfo)
		expectedChinfo := ChannelInfo{
			LastLane: 1,
			From:     clientAddr,
			To:       minerAddr,
			IDAddr:   paychIDAddr,
		}

		assert.Equal(t, expectedChinfo, *chinfo)
	})

	t.Run("errors if update lane doesn't exist", func(t *testing.T) {
		badAddr := spect.NewActorAddr(t, "nonexistent")
		lane, err := m.AllocateLane(badAddr)
		expErr := fmt.Sprintf("No state for /%s", badAddr.String())
		assert.EqualError(t, err, expErr)
		assert.Zero(t, lane)
	})
}

// CreateVoucher is called by a retrieval client
func TestManager_CreateVoucher(t *testing.T) {

}

// SaveVoucher is called by a retrieval provider
func TestManager_SaveVoucher(t *testing.T) {
	ctx := context.Background()
	paychUniqueAddr := spect.NewActorAddr(t, "abcd123")
	paychIDAddr := spect.NewIDAddr(t, 103)
	clientAddr := spect.NewIDAddr(t, 99)
	minerAddr := spect.NewIDAddr(t, 100)
	root := shared_testutil.GenerateCids(1)[0]
	cr := NewFakeChainReader(block.NewTipSetKey(root))
	proof := []byte("proof")
	amt := big.NewInt(300)
	sig := crypto.Signature{Type: crypto.SigTypeSecp256k1, Data: []byte("doesntmatter")}
	v := paych.SignedVoucher{
		Nonce:          2,
		TimeLockMax:    abi.ChainEpoch(12345),
		TimeLockMin:    abi.ChainEpoch(12346),
		Lane:           2,
		Amount:         amt,
		Signature:      &sig,
		SecretPreimage: []uint8{},
	}
	newV := v
	newV.Amount = abi.NewTokenAmount(500)

	t.Run("happy path", func(t *testing.T) {
		expVouchers := []*paych.SignedVoucher{&v, &newV}
		viewer, manager := saveVoucherSetup(ctx, t, root, cr)
		viewer.Views[root].AddActorWithState(paychUniqueAddr, clientAddr, minerAddr, paychIDAddr)
		for _, voucher := range expVouchers {
			resAmt, err := manager.SaveVoucher(paychUniqueAddr, voucher, proof)
			require.NoError(t, err)
			assert.Equal(t, voucher.Amount, resAmt)
		}
		has, err := manager.ChannelExists(paychUniqueAddr)
		require.NoError(t, err)
		assert.True(t, has)
		chinfo, err := manager.GetPaymentChannelInfo(paychUniqueAddr)
		require.NoError(t, err)
		require.NotNil(t, chinfo)
		for idx, voucher := range expVouchers {
			assert.True(t, reflect.DeepEqual(voucher, chinfo.Vouchers[idx].Voucher))
			assert.Equal(t, proof, chinfo.Vouchers[idx].Proof)

		}

	})

	t.Run("returns error if we try to save the same voucher", func(t *testing.T) {
		viewer, manager := saveVoucherSetup(ctx, t, root, cr)
		viewer.Views[root].AddActorWithState(paychUniqueAddr, clientAddr, minerAddr, paychIDAddr)
		resAmt, err := manager.SaveVoucher(paychUniqueAddr, &v, []byte("porkchops"))
		require.NoError(t, err)
		assert.Equal(t, amt, resAmt)

		resAmt, err = manager.SaveVoucher(paychUniqueAddr, &v, []byte("porkchops"))
		assert.EqualError(t, err, "voucher already saved: doesntmatter")
		assert.Equal(t, abi.NewTokenAmount(0), resAmt)
	})

	t.Run("returns error if marshaling fails", func(t *testing.T) {
		viewer, manager := saveVoucherSetup(ctx, t, root, cr)
		viewer.Views[root].AddActorWithState(paychUniqueAddr, clientAddr, minerAddr, address.Undef)
		resAmt, err := manager.SaveVoucher(paychUniqueAddr, &v, []byte("porkchops"))
		assert.EqualError(t, err, "cannot marshal undefined address")
		assert.Equal(t, abi.NewTokenAmount(0), resAmt)
	})

	t.Run("returns error if ResolveAddressAt fails", func(t *testing.T) {
		viewer, manager := saveVoucherSetup(ctx, t, root, cr)
		viewer.Views[root].ResolveAddressAtErr = errors.New("boom")
		viewer.Views[root].AddActorWithState(paychUniqueAddr, clientAddr, minerAddr, paychIDAddr)
		resAmt, err := manager.SaveVoucher(paychUniqueAddr, &v, []byte("porkchops"))
		assert.EqualError(t, err, "boom")
		assert.Equal(t, abi.NewTokenAmount(0), resAmt)
	})

	t.Run("returns error if cannot get actor state/parties", func(t *testing.T) {
		viewer, manager := saveVoucherSetup(ctx, t, root, cr)
		viewer.Views[root].AddActorWithState(paychUniqueAddr, clientAddr, minerAddr, paychIDAddr)
		viewer.Views[root].PaychActorPartiesErr = errors.New("boom")
		resAmt, err := manager.SaveVoucher(paychUniqueAddr, &v, []byte("porkchops"))
		assert.EqualError(t, err, "boom")
		assert.Equal(t, abi.NewTokenAmount(0), resAmt)
	})

	t.Run("returns err if cannot get head/tipset", func(t *testing.T) {
		cr2 := NewFakeChainReader(block.NewTipSetKey(root))
		cr2.GetTSErr = errors.New("kaboom")
		viewer, manager := saveVoucherSetup(ctx, t, root, cr2)
		viewer.Views[root].AddActorWithState(paychUniqueAddr, clientAddr, minerAddr, paychIDAddr)
		resAmt, err := manager.SaveVoucher(paychUniqueAddr, &v, []byte("porkchops"))
		assert.EqualError(t, err, "kaboom")
		assert.Equal(t, abi.NewTokenAmount(0), resAmt)
	})
}

func saveVoucherSetup(ctx context.Context, t *testing.T, root cid.Cid, cr *FakeChainReader) (*FakeStateViewer, *Manager) {
	testAPI := NewFakePaymentChannelAPI(ctx, t)
	ds := dss.MutexWrap(datastore.NewMapDatastore())
	viewer := makeStateViewer(t, root, nil)
	return viewer, NewManager(context.Background(), ds, testAPI, testAPI, viewer, cr)
}

func requireSetupPaymentChannel(t *testing.T, testAPI *FakePaymentChannelAPI, m *Manager) (address.Address, address.Address, address.Address, address.Address, uint64) {
	clientAddr := spect.NewIDAddr(t, 901)
	minerAddr := spect.NewIDAddr(t, 902)
	paychIDAddr := spect.NewIDAddr(t, 999)
	paychUniqueAddr := spect.NewActorAddr(t, "abcd123")
	blockHeight := uint64(1234)

	testAPI.StubCreatePaychActorMessage(clientAddr, minerAddr, paychIDAddr, paychUniqueAddr, builtin.MethodsInit.Exec, exitcode.Ok, blockHeight)

	err := m.CreatePaymentChannel(clientAddr, minerAddr)
	require.NoError(t, err)
	testAPI.Verify()
	return clientAddr, minerAddr, paychIDAddr, paychUniqueAddr, blockHeight
}

func makeStateViewer(t *testing.T, stateRoot cid.Cid, viewErr error) *FakeStateViewer {
	return &FakeStateViewer{
		Views: map[cid.Cid]*FakeStateView{stateRoot: NewFakeStateView(t, viewErr)}}
}
