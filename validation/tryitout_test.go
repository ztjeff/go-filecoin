package validation

import (
	"testing"

	vmtool "github.com/filecoin-project/chain-validation/pkg/validation"

)

func TestTryItOut(t *testing.T) {
	stateFactory := NewStateFactory()
	msgFactory := NewMessageFactory(stateFactory.Signer())
	vmtool.TryItOut(t, msgFactory, stateFactory)
}
