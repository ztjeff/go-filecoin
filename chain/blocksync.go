package chain

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"

	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-net"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"

	"github.com/filecoin-project/go-filecoin/types"
)

var log = logging.Logger("net/blocksync")

const BlockSyncProtocolID = "/fil/sync/blk"

func init() {
	cbor.RegisterCborType(BlockSyncRequest{})
	cbor.RegisterCborType(BlockSyncResponse{})
	cbor.RegisterCborType(BSTipSet{})

}

type BlockSyncService struct {
	cs *Store
}

type BSOptions struct {
	IncludeBlocks   bool
	IncludeMessages bool
}

func ParseBSOptions(optfield uint64) *BSOptions {
	return &BSOptions{
		IncludeBlocks:   optfield&(BSOptBlocks) != 0,
		IncludeMessages: optfield&(BSOptMessages) != 0,
	}
}

const (
	BSOptBlocks   = 1 << 0
	BSOptMessages = 1 << 1
)

type BlockSyncRequest struct {
	Start         []cid.Cid
	RequestLength uint64

	Options uint64
}

type BlockSyncResponse struct {
	Chain []*BSTipSet

	Status  uint
	Message string
}

type BSTipSet struct {
	Blocks []*types.Block
}

func NewBlockSyncService(cs *Store) *BlockSyncService {
	return &BlockSyncService{
		cs: cs,
	}
}

func (bss *BlockSyncService) HandleStream(s inet.Stream) {
	defer s.Close()
	log.Info("handling block sync request")

	var req BlockSyncRequest
	if err := ReadCborRPC(bufio.NewReader(s), &req); err != nil {
		log.Errorf("failed to read block sync request: %s", err)
		return
	}
	log.Infof("Read request: %v", req)

	resp, err := bss.processRequest(&req)
	if err != nil {
		log.Error("failed to process block sync request: ", err)
		return
	}

	if err := WriteCborRPC(s, resp); err != nil {
		log.Error("failed to write back response for handle stream: ", err)
		return
	}
}

func (bss *BlockSyncService) processRequest(req *BlockSyncRequest) (*BlockSyncResponse, error) {
	opts := ParseBSOptions(req.Options)
	chain, err := bss.collectChainSegment(req.Start, req.RequestLength, opts)
	if err != nil {
		log.Error("encountered error while responding to block sync request: ", err)
		return &BlockSyncResponse{
			Status: 203,
		}, nil
	}

	log.Info("process request success, chain: %v", chain)
	return &BlockSyncResponse{
		Chain:  chain,
		Status: 0,
	}, nil
}

func (bss *BlockSyncService) collectChainSegment(start []cid.Cid, length uint64, opts *BSOptions) ([]*BSTipSet, error) {
	var bstips []*BSTipSet
	cur := types.NewTipSetKey(start...)
	log.Infof("collectChainSegment - start: %v, length %d", start, length)
	for {
		var bst BSTipSet
		ts, err := bss.cs.GetTipSet(cur)
		if err != nil {
			return nil, err
		}
		log.Infof("got tipset: %v", ts.String())

		bst.Blocks = ts.ToSlice()
		bstips = append(bstips, &bst)

		tsH, err := ts.Height()
		if err != nil {
			return nil, err
		}

		if uint64(len(bstips)) >= length || tsH == 0 {
			log.Infof("returning chain segment len: %d", len(bstips))
			return bstips, nil
		}

		cur, err = ts.Parents()
		if err != nil {
			return nil, err
		}
	}
}

type NewStreamFunc func(ctx context.Context, p peer.ID, pids ...protocol.ID) (inet.Stream, error)

type BlockSync struct {
	bswap     *bitswap.Bitswap
	newStream NewStreamFunc

	syncPeersLk sync.Mutex
	syncPeers   map[peer.ID]struct{}
}

func NewBlockSyncClient(bswap *bitswap.Bitswap, newStreamF NewStreamFunc) *BlockSync {
	return &BlockSync{
		bswap:     bswap,
		newStream: newStreamF,
		syncPeers: make(map[peer.ID]struct{}),
	}
}

func (bs *BlockSync) getPeers() []peer.ID {
	bs.syncPeersLk.Lock()
	defer bs.syncPeersLk.Unlock()
	var out []peer.ID
	for p := range bs.syncPeers {
		out = append(out, p)
	}
	return out
}

func (bs *BlockSync) GetBlocks(ctx context.Context, tipset []cid.Cid, count int) ([]types.TipSet, error) {
	peers := bs.getPeers()
	perm := rand.Perm(len(peers))
	// TODO: round robin through these peers on error

	log.Infof("Request block %v, length %d, from peer %s", tipset, count, peers[perm[0]])
	req := &BlockSyncRequest{
		Start:         tipset,
		RequestLength: uint64(count),
		Options:       BSOptBlocks,
	}

	res, err := bs.sendRequestToPeer(ctx, peers[perm[0]], req)
	if err != nil {
		return nil, err
	}

	switch res.Status {
	case 0: // Success
		log.Infof("Got response for block %v, result %v", tipset, res.Chain[0])
		return bs.processBlocksResponse(req, res)
	case 101: // Partial Response
		panic("not handled")
	case 201: // req.Start not found
		return nil, fmt.Errorf("not found")
	case 202: // Go Away
		panic("not handled")
	case 203: // Internal Error
		return nil, fmt.Errorf("block sync peer errored: %s", res.Message)
	default:
		return nil, fmt.Errorf("unrecognized response code")
	}
}

func (bs *BlockSync) sendRequestToPeer(ctx context.Context, p peer.ID, req *BlockSyncRequest) (*BlockSyncResponse, error) {
	s, err := bs.newStream(net.WithNoDial(ctx, "should already have connection"), p, BlockSyncProtocolID)
	if err != nil {
		return nil, err
	}

	if err := WriteCborRPC(s, req); err != nil {
		return nil, err
	}

	var res BlockSyncResponse
	if err := ReadCborRPC(bufio.NewReader(s), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (bs *BlockSync) processBlocksResponse(req *BlockSyncRequest, res *BlockSyncResponse) ([]types.TipSet, error) {
	cur := res.Chain[0]

	curTs, err := types.NewTipSet(cur.Blocks...)
	if err != nil {
		return nil, err
	}
	out := []types.TipSet{curTs}
	for bi := 1; bi < len(res.Chain); bi++ {
		next := res.Chain[bi]

		nextTs, err := types.NewTipSet(next.Blocks...)
		if err != nil {
			return nil, err
		}

		curP, err := curTs.Parents()
		if err != nil {
			return nil, err
		}
		if !curP.Equals(nextTs.Key()) {
			return nil, fmt.Errorf("parents of tipset[%d] were not tipset[%d]", bi-1, bi)
		}

		out = append(out, nextTs)
		cur = next
	}
	log.Infof("processBlockResponse returning %v", out)
	return out, nil
}

func (bs *BlockSync) GetBlock(ctx context.Context, c cid.Cid) (*types.Block, error) {
	sb, err := bs.bswap.GetBlock(ctx, c)
	if err != nil {
		return nil, err
	}

	return types.DecodeBlock(sb.RawData())
}

func (bs *BlockSync) AddPeer(p peer.ID) {
	bs.syncPeersLk.Lock()
	defer bs.syncPeersLk.Unlock()
	bs.syncPeers[p] = struct{}{}
}

const MessageSizeLimit = 1 << 20

func WriteCborRPC(w io.Writer, obj interface{}) error {
	data, err := cbor.DumpObject(obj)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

type ByteReader interface {
	io.Reader
	io.ByteReader
}

func ReadCborRPC(r ByteReader, out interface{}) error {
	log.Info("Starting Request Read")
	defer log.Info("Completed Request Read")
	return cbor.DecodeReader(r, out)
}
