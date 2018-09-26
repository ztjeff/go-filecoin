package impl

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"time"

	logging "gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log"
	writer "gx/ipfs/QmRREK2CAZ5Re2Bd9zZFG6FeYDppUWt5cMgsoUEp3ktgSr/go-log/writer"
	manet "gx/ipfs/QmV6FjemM1K8oXjrvuq3wuVWWoU2TLDPmNnKrxHzY3v6Ai/go-multiaddr-net"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
)

var log = logging.Logger("api/impl")
var LogStreamJoinEvent = "LogStreamJoin"
var LogStreamLeaveEvent = "LogStreamLeave"

type nodeLog struct {
	api *nodeAPI
}

func newNodeLog(api *nodeAPI) *nodeLog {
	return &nodeLog{api: api}
}

func (api *nodeLog) Tail(ctx context.Context) io.Reader {
	r, w := io.Pipe()
	go func() {
		defer w.Close() // nolint: errcheck
		<-ctx.Done()
	}()

	writer.WriterGroup.AddWriter(w)

	return r
}

func (api *nodeLog) Stream(ctx context.Context, maddr ma.Multiaddr) error {
	nodeDetails, err := api.api.ID().Details()
	if err != nil {
		return err
	}
	peerID := nodeDetails.ID
	// Get the nodes nickname.
	nodeNic, err := api.api.Config().Get("stats.nodeNic")
	if err != nil {
		return err
	}

	// connection the logs will stream on
	mconn, err := manet.Dial(maddr)
	if err != nil {
		return err
	}
	defer mconn.Close()
	wconn := bufio.NewWriter(mconn)

	r, w := io.Pipe()
	go func() {
		// node leaves a connection
		defer w.Close() // nolint: errcheck
		defer r.Close()
		<-ctx.Done()
		ctx = log.Start(ctx, LogStreamLeaveEvent)
		log.SetTag(ctx, "peerID", peerID)
		log.Finish(ctx)
		time.Sleep(2 * time.Second)
	}()

	writer.WriterGroup.AddWriter(w)

	// node joins a connection
	ctx = log.Start(ctx, LogStreamJoinEvent)
	log.SetTag(ctx, "peerID", peerID)
	log.Finish(ctx)

	// Lets make a crappy filter
	filterR, filterW := io.Pipe()
	go func() {
		defer filterR.Close()
		defer filterW.Close()
		<-ctx.Done()
	}()

	filterDecoder := json.NewDecoder(r)
	filterEncoder := json.NewEncoder(filterW)
	go func() {
		for {
			if ctx.Err() != nil {
				log.Warningf("filter context error: %v", ctx.Err())
				break
			}
			var event map[string]interface{}
			filterDecoder.Decode(&event)
			if event == nil {
				continue
			}
			// "filter"
			// add things to the event log here
			event["peerName"] = nodeNic
			event["peerID"] = peerID
			filterEncoder.Encode(event)
		}
	}()

	_, err = wconn.ReadFrom(filterR)
	if err != nil {
		return err
	}
	// flush the rest of the events that may be in the pipe before the defered close
	wconn.Flush()

	return nil
}
