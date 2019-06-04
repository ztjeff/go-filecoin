package consensus

import (
	"encoding/binary"

	"github.com/minio/sha256-simd"

	"github.com/filecoin-project/go-filecoin/types"
)

// CreateChallengeSeed creates/recreates the block challenge for purposes of validation.
//   TODO -- in general this won't work with only the base tipset.
//     We'll potentially need some chain manager utils, similar to
//     the State function, to sample further back in the chain.
func CreateChallengeSeed(parents types.TipSet, nullBlkCount uint64) (types.PoStChallengeSeed, error) {
	smallest, err := parents.MinTicket()
	if err != nil {
		return types.PoStChallengeSeed{}, err
	}

	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, nullBlkCount)
	buf = append(smallest, buf[:n]...)

	h := sha256.Sum256(buf)
	return h, nil
}
