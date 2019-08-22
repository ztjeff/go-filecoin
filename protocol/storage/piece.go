package storage

import (
	"io"

	"github.com/ipfs/go-cid"
)

type Piece interface {
	Cid() cid.Cid
	Size() uint64
	Fetch() error
	Reader() io.Reader
}
