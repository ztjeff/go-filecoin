package consensus

import (
	"github.com/filecoin-project/go-filecoin/actor/builtin/miner"
)

// TODO none of these parameters are chosen correctly
// with respect to analysis under a security model:
// https://github.com/filecoin-project/go-filecoin/issues/1846

// ECV is the constant V defined in the EC spec.
const ECV uint64 = 10

// ECPrM is the power ratio magnitude defined in the EC spec.
const ECPrM uint64 = 100

// AncestorRoundsNeeded is the number of rounds of the ancestor chain needed
// to process all state transitions.
//
// TODO: If the following PR is merged - and the network doesn't define a
// largest sector size - this constant will need to be reconsidered.
// https://github.com/filecoin-project/specs/pull/318
const AncestorRoundsNeeded = miner.LargestSectorSizeProvingPeriodBlocks + miner.LargestSectorGenerationAttackThresholdBlocks
