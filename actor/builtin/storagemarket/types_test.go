package storagemarket

import (
	cbor "gx/ipfs/QmV6BQ6fFCf9eFHDuRxvguvqfKLZtZrxthgZvDfRCs4tMN/go-ipld-cbor"
	"testing"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/stretchr/testify/assert"
)

func TestAskSetMarshaling(t *testing.T) {
	assert := assert.New(t)
	addrGetter := address.NewForTestGetter()

	as := make(AskSet)
	ask4 := &Ask{ID: 4, Owner: addrGetter(), Price: types.NewAttoFILFromFIL(19), Size: types.NewBytesAmount(105)}
	ask5 := &Ask{ID: 5, Owner: addrGetter(), Price: types.NewAttoFILFromFIL(909), Size: types.NewBytesAmount(435)}
	as[4] = ask4
	as[5] = ask5

	data, err := cbor.DumpObject(as)
	assert.NoError(err)

	var asout AskSet
	assert.NoError(cbor.DecodeInto(data, &asout))
	assert.Len(asout, 2)
	ask4out, ok := as[4]
	assert.True(ok)
	assert.Equal(ask4, ask4out)
	ask5out, ok := as[5]
	assert.True(ok)
	assert.Equal(ask5, ask5out)
}
