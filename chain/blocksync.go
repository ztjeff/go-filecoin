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

const BlockSyncProtocolID = "/fil/sync/blk"

func init() {
	cbor.RegisterCborType(BlockSyncRequest{})
	cbor.RegisterCborType(BlockSyncResponse{})
	cbor.RegisterCborType(BSTipSet{})

}

type BlockSyncService struct {
	cs  *Store
	log logging.EventLogger
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

func (bsr BlockSyncRequest) String() string {
	opts := ParseBSOptions(bsr.Options)
	return fmt.Sprintf("Start: %s Length: %d Blks: %t Msgs: %t", bsr.Start, bsr.RequestLength, opts.IncludeBlocks, opts.IncludeMessages)
}

type BlockSyncResponse struct {
	Chain []*BSTipSet

	Status  uint
	Message string
}

func (bsr BlockSyncResponse) String() string {
	return fmt.Sprintf("Head: %s Status: %d Message: %s", bsr.Chain[0].Blocks[0].Cid(), bsr.Status, bsr.Message)
}

type BSTipSet struct {
	Blocks []*types.Block
}

func NewBlockSyncService(cs *Store) *BlockSyncService {
	return &BlockSyncService{
		cs:  cs,
		log: logging.Logger("blocksync/service"),
	}
}

func (bss *BlockSyncService) HandleStream(s inet.Stream) {
	defer s.Close()

	var req BlockSyncRequest
	if err := ReadCborRPC(bufio.NewReader(s), &req); err != nil {
		bss.log.Errorf("failed to read block sync request: %s", err)
		return
	}

	resp, err := bss.processRequest(&req)
	if err != nil {
		bss.log.Error("failed to process block sync request: ", err)
		return
	}

	if err := WriteCborRPC(s, resp); err != nil {
		bss.log.Error("failed to write back response for handle stream: ", err)
		return
	}
}

func (bss *BlockSyncService) processRequest(req *BlockSyncRequest) (*BlockSyncResponse, error) {
	bss.log.Infof("processing request: %s", req)

	opts := ParseBSOptions(req.Options)
	chain, err := bss.collectChainSegment(req.Start, req.RequestLength, opts)
	if err != nil {
		bss.log.Error("encountered error while responding to block sync request: ", err)
		return &BlockSyncResponse{
			Status: 203,
		}, nil
	}

	return &BlockSyncResponse{
		Chain:  chain,
		Status: 0,
	}, nil
}

func (bss *BlockSyncService) collectChainSegment(start []cid.Cid, length uint64, opts *BSOptions) ([]*BSTipSet, error) {
	var bstips []*BSTipSet
	cur := types.NewTipSetKey(start...)
	bss.log.Infof("collecting chain length %d starting at %s", length, start)
	for {
		var bst BSTipSet
		ts, err := bss.cs.GetTipSet(cur)
		if err != nil {
			bss.log.Warningf("collect chain failed to find %s in chain store", cur.String())
			return nil, err
		}

		bst.Blocks = ts.ToSlice()
		bstips = append(bstips, &bst)

		tsH, err := ts.Height()
		if err != nil {
			return nil, err
		}

		if uint64(len(bstips)) >= length || tsH == 0 {
			bss.log.Infof("collecting chain complete returning chain length %d starting at %s", length, bstips[0].Blocks[0].Cid())
			return bstips, nil
		}

		cur, err = ts.Parents()
		if err != nil {
			return nil, err
		}
	}
}

type RequestHandler interface {
	SendRequest(context.Context, peer.ID, *BlockSyncRequest) (*BlockSyncResponse, error)
}

type NewStreamFunc func(ctx context.Context, p peer.ID, pids ...protocol.ID) (inet.Stream, error)

type BlockSyncRequestHandler struct {
	log       logging.EventLogger
	newStream NewStreamFunc
}

func NewBlockSyncRequestHandler(newStreamF NewStreamFunc) *BlockSyncRequestHandler {
	return &BlockSyncRequestHandler{
		log:       logging.Logger("blocksync/requestHandler"),
		newStream: newStreamF,
	}
}

func (bsrh *BlockSyncRequestHandler) SendRequest(ctx context.Context, p peer.ID, req *BlockSyncRequest) (*BlockSyncResponse, error) {
	bsrh.log.Infof("Sending request: %s to peer %s", req, p.Pretty())

	s, err := bsrh.newStream(net.WithNoDial(ctx, "should already have connection"), p, BlockSyncProtocolID)
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

	bsrh.log.Infof("Received response: %s from peer %s", res, p.Pretty())

	return &res, nil
}

type BlockSync struct {
	RequestHandler

	bswap *bitswap.Bitswap

	syncPeersLk sync.Mutex
	syncPeers   map[peer.ID]struct{}

	log logging.EventLogger
}

func NewBlockSyncClient(bswap *bitswap.Bitswap, rh RequestHandler) *BlockSync {
	return &BlockSync{
		bswap:          bswap,
		RequestHandler: rh,
		syncPeers:      make(map[peer.ID]struct{}),
		log:            logging.Logger("blocksync/client"),
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

func (bs *BlockSync) FetchTipSets(ctx context.Context, tsKey types.TipSetKey, recur int) ([]types.TipSet, error) {
	peers := bs.getPeers()
	perm := rand.Perm(len(peers))
	// TODO: round robin through these peers on error
	req := &BlockSyncRequest{
		Start:         tsKey.ToSlice(),
		RequestLength: uint64(recur),
		Options:       BSOptBlocks,
	}

	res, err := bs.SendRequest(ctx, peers[perm[0]], req)
	if err != nil {
		return nil, err
	}

	switch res.Status {
	case 0: // Success
		return bs.processBlocksResponse(res)
	case 101: // Partial Response
		err := fmt.Errorf("partial response not handled")
		bs.log.Errorf("Received response from peer %s, Error: %s", peers[perm[0]], err.Error())
		return nil, err
	case 201: // req.Start not found
		err := fmt.Errorf("not found")
		bs.log.Errorf("Received response from peer %s, Error: %s", peers[perm[0]], err.Error())
		return nil, err
	case 202: // Go Away
		err := fmt.Errorf("go away")
		bs.log.Errorf("Received response from peer %s, Error: %s", peers[perm[0]], err.Error())
		return nil, err
	case 203: // Internal Error
		return nil, fmt.Errorf("Received response from peer %s, Error: %s", peers[perm[0]], res.Message)
	default:
		return nil, fmt.Errorf("unrecognized response code")
	}
}

/*
func (bs *BlockSync) sendRequestToPeer(ctx context.Context, p peer.ID, req *BlockSyncRequest) (*BlockSyncResponse, error) {
	bs.log.Infof("Sending request: %s to peer %s", req, p.Pretty())

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

	bs.log.Infof("Received response: %s from peer %s", res, p.Pretty())

	return &res, nil
}
*/

func (bs *BlockSync) processBlocksResponse(res *BlockSyncResponse) ([]types.TipSet, error) {
	bs.log.Infof("processing response %s", res)
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
	return cbor.DecodeReader(r, out)
}
