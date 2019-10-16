package validation

import (
	"math/big"
	"testing"

	"github.com/filecoin-project/chain-validation/pkg/chain"
	"github.com/filecoin-project/chain-validation/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
)

func TestMessageFactory(t *testing.T) {
	factory := NewMessageFactory()
	p := chain.NewMessageProducer(factory)

	require.NoError(t, p.Transfer(state.NetworkAddress, state.BurntFundsAddress, big.NewInt(1)))

	messages := p.Messages()
	assert.Equal(t,1, len(messages))
	msg := messages[0].(*types.Message)
	assert.Equal(t, address.NetworkAddress, msg.From)
	assert.Equal(t, address.BurntFundsAddress, msg.To)
	assert.Equal(t, types.NewAttoFIL(big.NewInt(1)), msg.Value)
}
