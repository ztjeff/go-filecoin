package piece

import (
	"context"
	"fmt"
	"io"

	"github.com/ipfs/go-blockservice"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	offline "github.com/ipfs/go-ipfs-exchange-offline"

	"github.com/ipfs/go-cid"
	chunk "github.com/ipfs/go-ipfs-chunker"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	imp "github.com/ipfs/go-unixfs/importer"
	uio "github.com/ipfs/go-unixfs/io"
)

// Service is a service for accessing the merkledag
type Service struct {
	// The two exchanges disambiguate which operations should load data from the network (exchange).
	// exchds - will load data from other connected peers
	// localds - will only read content from the local dag/data/blockstore
	exchds  ipld.DAGService
	localds ipld.DAGService
}

// NewService creates a Service suitable for reading and writing pieces and fetching pieces from the network.
func NewService(bstore blockstore.Blockstore, exch exchange.Interface) *Service {
	return &Service{
		exchds:  merkledag.NewDAGService(blockservice.New(bstore, exch)),
		localds: merkledag.NewDAGService(blockservice.New(bstore, offline.Exchange(bstore))),
	}
}

// Size returns the file size for a given Cid
func (ps *Service) Size(ctx context.Context, c cid.Cid) (uint64, error) {
	fnode, err := ps.localds.Get(ctx, c)
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
func (ps *Service) Read(ctx context.Context, c cid.Cid) (io.ReadSeeker, error) {
	data, err := ps.localds.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	return uio.NewDagReader(ctx, data, ps.localds)
}

// Write adds data from an io stream to the merkledag and returns the Cid
// of the given data
func (ps *Service) Write(ctx context.Context, data io.Reader) (ipld.Node, error) {
	bufds := ipld.NewBufferedDAG(ctx, ps.localds)

	spl := chunk.DefaultSplitter(data)

	nd, err := imp.BuildDagFromReader(bufds, spl)
	if err != nil {
		return nil, err
	}
	return nd, bufds.Commit()
}

// Fetch will retrieve the piece data over the network and store it in the DAG
func (ps *Service) Fetch(ctx context.Context, cid cid.Cid) error {
	return merkledag.FetchGraph(ctx, cid, ps.exchds)
}
