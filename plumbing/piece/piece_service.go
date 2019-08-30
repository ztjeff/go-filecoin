package piece

import (
	"context"
	"fmt"
	"io"

	"github.com/ipfs/go-blockservice"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"

	"github.com/ipfs/go-cid"
	chunk "github.com/ipfs/go-ipfs-chunker"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	imp "github.com/ipfs/go-unixfs/importer"
	uio "github.com/ipfs/go-unixfs/io"
)

// PieceService is a service for accessing the merkledag
type PieceService struct {
	dserv ipld.DAGService
}

// NewPieceService creates a PieceService with a given DAGService
func NewPieceService(bstore blockstore.Blockstore, exch exchange.Interface) *PieceService {
	bserv := blockservice.New(bstore, exch)
	ds := merkledag.NewDAGService(bserv)

	return &PieceService{
		dserv: ds,
	}
}

// GetFileSize returns the file size for a given Cid
func (ps *PieceService) GetFileSize(ctx context.Context, c cid.Cid) (uint64, error) {
	fnode, err := ps.dserv.Get(ctx, c)
	if err != nil {
		return 0, err
	}
	switch n := fnode.(type) {
	case *merkledag.ProtoNode:
		return unixfs.DataSize(n.Data())
	case *merkledag.RawNode:
		return n.Size()
	default:
		return 0, fmt.Errorf("unrecognized node type: %T", fnode)
	}
}

// Read returns an iostream with a piece of data stored on the merkeldag with
// the given cid.
//
// TODO: this goes back to 'how is data stored and referenced'
// For now, lets just do things the ipfs way.
// https://github.com/filecoin-project/specs/issues/136
func (ps *PieceService) Read(ctx context.Context, c cid.Cid) (uio.DagReader, error) {
	data, err := ps.dserv.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	return uio.NewDagReader(ctx, data, ps.dserv)
}

// Write adds data from an io stream to the merkledag and returns the Cid
// of the given data
func (ps *PieceService) Write(ctx context.Context, data io.Reader) (ipld.Node, error) {
	bufds := ipld.NewBufferedDAG(ctx, ps.dserv)

	spl := chunk.DefaultSplitter(data)

	nd, err := imp.BuildDagFromReader(bufds, spl)
	if err != nil {
		return nil, err
	}
	return nd, bufds.Commit()
}

func (ps *PieceService) Fetch(ctx context.Context, cid cid.Cid) error {
	return merkledag.FetchGraph(ctx, cid, ps.dserv)
}
