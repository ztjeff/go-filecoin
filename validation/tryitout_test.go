package validation

import (
	"testing"

	vmtool "github.com/filecoin-project/chain-validation/pkg/validation"

)

func TestTryItOut(t *testing.T) {
	stateFactory := StateFactory{}
	msgFactory := &MessageFactory{}
	storageFactory := &StorageFactory{}
	vmtool.TryItOut(t, msgFactory, stateFactory, storageFactory)
}
