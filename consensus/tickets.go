package consensus

import (
	"context"
	"math/big"

	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/state"
	"github.com/filecoin-project/go-filecoin/types"
)

var ticketDomain *big.Int

func init() {
	ticketDomain = &big.Int{}
	// The size of the ticket domain must equal the size of the Signature (ticket) generated.
	// Currently this is a secp256k1.Sign signature, which is 65 bytes.
	ticketDomain.Exp(big.NewInt(2), big.NewInt(65*8), nil)
	ticketDomain.Sub(ticketDomain, big.NewInt(1))
}

// TicketSigner is an interface for a test signer that can create tickets.
type TicketSigner interface {
	GetAddressForPubKey(pk []byte) (address.Address, error)
	SignBytes(data []byte, signerAddr address.Address) (types.Signature, error)
}

// CreateTicket computes a valid ticket.
// 	params:  proof  []byte, the proof to sign
// 			 signerPubKey []byte, the public key for the signer. Must exist in the signer
//      	 signer, implements TicketSigner interface. Must have signerPubKey in its keyinfo.
//  returns:  types.Signature ( []byte ), error
func CreateTicket(proof types.PoStProof, signerPubKey []byte, signer TicketSigner) (types.Signature, error) {

	var ticket types.Signature

	signerAddr, err := signer.GetAddressForPubKey(signerPubKey)
	if err != nil {
		return ticket, errors.Wrap(err, "could not get address for signerPubKey")
	}
	buf := append(proof[:], signerAddr.Bytes()...)
	// Don't hash it here; it gets hashed in walletutil.Sign
	return signer.SignBytes(buf[:], signerAddr)
}

// CompareTicketPower abstracts the actual comparison logic so it can be used by some test
// helpers
func CompareTicketPower(ticket types.Signature, minerPower *types.BytesAmount, totalPower *types.BytesAmount) bool {
	lhs := &big.Int{}
	lhs.SetBytes(ticket)
	lhs.Mul(lhs, totalPower.BigInt())
	rhs := &big.Int{}
	rhs.Mul(minerPower.BigInt(), ticketDomain)
	return lhs.Cmp(rhs) < 0
}

// IsWinningTicket fetches miner power & total power, returns true if it's a winning ticket, false if not,
//    errors out if minerPower or totalPower can't be found.
//    See https://github.com/filecoin-project/specs/blob/master/expected-consensus.md
//    for an explanation of the math here.
func IsWinningTicket(ctx context.Context, bs blockstore.Blockstore, ptv PowerTableView, st state.Tree,
	ticket types.Signature, miner address.Address) (bool, error) {

	totalPower, err := ptv.Total(ctx, st, bs)
	if err != nil {
		return false, errors.Wrap(err, "Couldn't get totalPower")
	}

	minerPower, err := ptv.Miner(ctx, st, bs, miner)
	if err != nil {
		return false, errors.Wrap(err, "Couldn't get minerPower")
	}

	return CompareTicketPower(ticket, minerPower, totalPower), nil
}
