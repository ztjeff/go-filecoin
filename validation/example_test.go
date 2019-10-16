package validation

import (
	"testing"

	vmtool "github.com/filecoin-project/chain-validation/pkg/validation"
)

func TestExample(t *testing.T) {
	stateFactory := NewStateFactory()
	msgFactory := NewMessageFactory(stateFactory.Signer())
	vmtool.Example(t, msgFactory, stateFactory)
}
