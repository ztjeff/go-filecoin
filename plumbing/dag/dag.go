package dag

import (
	"context"
	"fmt"
	"io"

	"github.com/ipfs/go-cid"
	chunk "github.com/ipfs/go-ipfs-chunker"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	imp "github.com/ipfs/go-unixfs/importer"
	uio "github.com/ipfs/go-unixfs/io"
)

// DAG is a service for accessing the merkledag
type DAG struct {
	dserv ipld.DAGService
}

// NewDAG creates a DAG with a given DAGService
func NewDAG(dserv ipld.DAGService) *DAG {
	return &DAG{
		dserv: dserv,
	}
}

// GetFileSize returns the file size for a given Cid
func (dag *DAG) GetFileSize(ctx context.Context, c cid.Cid) (uint64, error) {
	fnode, err := dag.dserv.Get(ctx, c)
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
func (dag *DAG) Read(ctx context.Context, c cid.Cid) (uio.DagReader, error) {
	data, err := dag.dserv.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	return uio.NewDagReader(ctx, data, dag.dserv)
}

// Write adds data from an io stream to the merkledag and returns the Cid
// of the given data
func (dag *DAG) Write(ctx context.Context, data io.Reader) (ipld.Node, error) {
	bufds := ipld.NewBufferedDAG(ctx, dag.dserv)

	spl := chunk.DefaultSplitter(data)

	nd, err := imp.BuildDagFromReader(bufds, spl)
	if err != nil {
		return nil, err
	}
	return nd, bufds.Commit()
}

func (dag *DAG) Fetch(ctx context.Context, cid cid.Cid) error {
	return merkledag.FetchGraph(ctx, cid, dag.dserv)
}
