package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/go-filecoin/proofs"
	"github.com/filecoin-project/go-filecoin/proofs/sectorbuilder"
	"github.com/filecoin-project/go-filecoin/proofs/verification"
	tf "github.com/filecoin-project/go-filecoin/testhelpers/testflags"
	"github.com/filecoin-project/go-filecoin/types"

	"github.com/stretchr/testify/require"
)

// MaxTimeToSealASector represents the maximum amount of time the test should
// wait for a sector to be sealed. Seal performance varies depending on the
// computer, so we need to select a value which works for slow (CircleCI OSX
// build containers) and fast (developer machines) alike.
const MaxTimeToSealASector = time.Second * 360000

// MaxTimeToGenerateSectorPoSt represents the maximum amount of time the test
// should wait for a proof-of-spacetime to be generated for a sector.
const MaxTimeToGenerateSectorPoSt = time.Second * 360000

func TestSectorBuilder(t *testing.T) {
	tf.UnitTest(t)

	t.Run("proof-of-spacetime generation and verification", func(t *testing.T) {
		h := NewBuilder(t).Build()
		defer h.Close()

		inputBytes := RequireRandomBytes(t, h.MaxBytesPerSector.Uint64())
		ref, size, reader, err := h.CreateAddPieceArgs(inputBytes)
		require.NoError(t, err)

		sectorID, err := h.SectorBuilder.AddPiece(context.Background(), ref, size, reader)
		require.NoError(t, err)

		timeout := time.After(MaxTimeToSealASector + MaxTimeToGenerateSectorPoSt)

		select {
		case val := <-h.SectorBuilder.SectorSealResults():
			require.NoError(t, val.SealingErr)
			require.Equal(t, sectorID, val.SealingResult.SectorID)

			sres, serr := (&verification.RustVerifier{}).VerifySeal(verification.VerifySealRequest{
				CommD:      val.SealingResult.CommD,
				CommR:      val.SealingResult.CommR,
				CommRStar:  val.SealingResult.CommRStar,
				Proof:      val.SealingResult.Proof,
				ProverID:   sectorbuilder.AddressToProverID(h.MinerAddr),
				SectorID:   sectorbuilder.SectorIDToBytes(val.SealingResult.SectorID),
				SectorSize: types.TwoHundredFiftySixMiBSectorSize,
			})
			require.NoError(t, serr, "seal proof-verification produced an error")
			require.True(t, sres.IsValid, "seal proof was not valid")

			// TODO: This should be generates from some standard source of
			// entropy, e.g. the blockchain
			challengeSeed := types.PoStChallengeSeed{1, 2, 3}

			sortedCommRs := proofs.NewSortedCommRs(val.SealingResult.CommR)

			fmt.Println()
			fmt.Println("POST BEGIN")

			// generate a proof-of-spacetime
			gres, gerr := h.SectorBuilder.GeneratePoSt(sectorbuilder.GeneratePoStRequest{
				SortedCommRs:  sortedCommRs,
				ChallengeSeed: challengeSeed,
			})
			require.NoError(t, gerr)

			fmt.Println("POST END")

			// verify the proof-of-spacetime
			vres, verr := (&verification.RustVerifier{}).VerifyPoSt(verification.VerifyPoStRequest{
				ChallengeSeed: challengeSeed,
				SortedCommRs:  sortedCommRs,
				Faults:        gres.Faults,
				Proofs:        gres.Proofs,
				SectorSize:    types.TwoHundredFiftySixMiBSectorSize,
			})

			require.NoError(t, verr)
			require.True(t, vres.IsValid)
		case <-timeout:
			t.Fatalf("timed out waiting for seal to complete")
		}
	})
}
