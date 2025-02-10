package lightconsensusrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	consensus "github.com/anyproto/any-sync-consensusnode"
	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/consensus/consensusproto"
	"github.com/anyproto/any-sync/consensus/consensusproto/consensuserr"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/nodeconf"
	"github.com/anyproto/any-sync/util/cidutil"
	"go.uber.org/zap"
)

const CName = "light.consensus.consensusrpc"

// TODO: Maybe use original RPC and just implement Database Interface
var (
	log    = logger.NewNamed(CName)
	okResp = &consensusproto.Ok{}
)

type dbSrv interface {
	// AddLog adds new log db
	AddLog(ctx context.Context, log consensus.Log) (err error)
	// DeleteLog deletes the log
	DeleteLog(ctx context.Context, logId string) error
	// AddRecord adds new record to existing log
	// returns consensuserr.ErrConflict if record didn't match or log not found
	AddRecord(ctx context.Context, logId string, record consensus.Record) (err error)
	// FetchLog gets log by id
	FetchLog(ctx context.Context, logId string) (log consensus.Log, err error)
}

type recordUpdate struct {
	logId  string
	record consensus.Record
}

type lightConsensusRpc struct {
	db       dbSrv
	account  accountservice.Service
	nodeconf nodeconf.Service
	dRPC     server.DRPCServer

	watchers sync.Map // map[chan *recordUpdate]struct{}
}

func New() *lightConsensusRpc {
	return &lightConsensusRpc{}
}

//
// App Component
//

func (c *lightConsensusRpc) Init(a *app.App) error {
	log.Info("call Init")

	c.db = app.MustComponent[dbSrv](a)
	c.account = app.MustComponent[accountservice.Service](a)
	c.nodeconf = app.MustComponent[nodeconf.Service](a)
	c.dRPC = app.MustComponent[server.DRPCServer](a)

	return nil
}

func (c *lightConsensusRpc) Name() (name string) {
	return CName
}

//
// App Component Runnable
//

func (c *lightConsensusRpc) Run(_ context.Context) error {
	log.Info("call Run")

	// TODO: Create a ticket to move call of this in original code to the Run stage, to avoid call any logic in Init
	return consensusproto.DRPCRegisterConsensus(c.dRPC, c)
}

func (c *lightConsensusRpc) Close(ctx context.Context) error {
	log.Info("call Close")

	return nil
}

//
// Component RPC
//

func (c *lightConsensusRpc) LogAdd(ctx context.Context, req *consensusproto.LogAddRequest) (resp *consensusproto.Ok, err error) {
	log.InfoCtx(ctx, "call LogAdd",
		zap.String("logId", req.LogId),
		zap.String("recordId", req.Record.Id),
		zap.String("recordPayload", string(req.Record.Payload)),
	)

	if err = c.checkWrite(ctx); err != nil {
		return
	}

	if req.GetRecord() == nil || !cidutil.VerifyCid(req.Record.Payload, req.Record.Id) {
		err = consensuserr.ErrInvalidPayload
		return
	}

	// we don't sign the first record because it affects the id, but we sign the following records as a confirmation that the chain is valid and the record added from a valid source
	l := consensus.Log{
		Id: req.LogId,
		Records: []consensus.Record{
			{
				Id:      req.Record.Id,
				Payload: req.Record.Payload,
				Created: time.Now(),
			},
		},
	}
	if err = c.db.AddLog(ctx, l); err != nil {
		return
	}
	return okResp, nil
}

func (c *lightConsensusRpc) LogDelete(ctx context.Context, req *consensusproto.LogDeleteRequest) (resp *consensusproto.Ok, err error) {
	log.InfoCtx(ctx, "call LogDelete", zap.String("request", req.String()))

	if err = c.checkWrite(ctx); err != nil {
		return
	}

	if err = c.db.DeleteLog(ctx, req.LogId); err != nil {
		return
	}
	return okResp, nil
}

func (c *lightConsensusRpc) RecordAdd(ctx context.Context, req *consensusproto.RecordAddRequest) (resp *consensusproto.RawRecordWithId, err error) {
	log.InfoCtx(ctx, "call RecordAdd", zap.String("request", req.String()))

	if err = c.checkWrite(ctx); err != nil {
		return
	}

	// unmarshal payload as a consensus record
	rec := &consensusproto.Record{}
	if e := rec.Unmarshal(req.Record.Payload); e != nil {
		err = consensuserr.ErrInvalidPayload
		return
	}

	// set an accept time
	createdTime := time.Now()
	req.Record.AcceptorTimestamp = createdTime.Unix()

	// sign a record
	req.Record.AcceptorIdentity = c.account.Account().SignKey.GetPublic().Storage()
	if req.Record.AcceptorSignature, err = c.account.Account().SignKey.Sign(req.Record.Payload); err != nil {
		log.Warn("can't sign payload", zap.Error(err))
		err = consensuserr.ErrUnexpected
		return
	}
	// marshal with identity and sign
	payload, err := req.Record.Marshal()
	if err != nil {
		log.Warn("can't marshal payload", zap.Error(err))
		err = consensuserr.ErrUnexpected
		return
	}

	// create id
	id, err := cidutil.NewCidFromBytes(payload)
	if err != nil {
		log.Warn("can't make payload cid", zap.Error(err))
		err = consensuserr.ErrUnexpected
		return
	}

	// add to db
	record := consensus.Record{
		Id:      id,
		PrevId:  rec.PrevId,
		Payload: payload,
		Created: createdTime,
	}

	if err = c.db.AddRecord(ctx, req.LogId, record); err != nil {
		return
	}

	// Add broadcast for LogWatch
	c.broadcast(&recordUpdate{
		logId:  req.LogId,
		record: record,
	})

	return &consensusproto.RawRecordWithId{
		Payload: payload,
		Id:      id,
	}, nil
}

// LogWatch watches for new records in the log
// Receive: logIDs that need to be watched or unwatched
// Send: initially all records for logID and records that will be added to the selected log in the future
func (c *lightConsensusRpc) LogWatch(stream consensusproto.DRPCConsensus_LogWatchStream) error {
	log.Info("call LogWatch")

	ctx := stream.Context()
	if err := c.checkRead(ctx); err != nil {
		return err
	}

	var (
		watchMu     sync.RWMutex
		watchedLogs = make(map[string]struct{})
		updates     = make(chan *recordUpdate, 100)
		done        = make(chan struct{})
	)

	c.addWatcher(updates)
	defer func() {
		close(done)
		c.removeWatcher(updates)
	}()

	sendUpdate := func(update *recordUpdate) error {
		log.Info("send update", zap.String("logId", update.logId), zap.String("recordId", update.record.Id))
		return stream.Send(&consensusproto.LogWatchEvent{
			LogId:   update.logId,
			Records: recordsToProto([]consensus.Record{update.record}),
		})
	}

	handleReceive := func(req *consensusproto.LogWatchRequest) error {
		watchMu.Lock()
		defer watchMu.Unlock()

		for _, logId := range req.UnwatchIds {
			log.Info("unwatch", zap.String("logId", logId))
			delete(watchedLogs, logId)
		}

		for _, logId := range req.WatchIds {
			log.Info("watch", zap.String("logId", logId))

			// TODO, Do we need to ignore already watched logs?
			if _, watching := watchedLogs[logId]; watching {
				log.Warn("already watching", zap.String("logId", logId))
				continue
			}

			watchedLogs[logId] = struct{}{}

			logEntry, err := c.db.FetchLog(ctx, logId)
			if err != nil {
				if errors.Is(err, consensuserr.ErrLogNotFound) {
					return stream.Send(&consensusproto.LogWatchEvent{
						LogId: logId,
						Error: &consensusproto.Err{Error: consensusproto.ErrCodes_LogNotFound},
					})
				}

				return fmt.Errorf("fetch log failed: %w", err)
			}

			if errSend := stream.Send(&consensusproto.LogWatchEvent{
				LogId:   logEntry.Id,
				Records: recordsToProto(logEntry.Records),
			}); errSend != nil {
				return fmt.Errorf("send initial state failed: %w", errSend)
			}
		}

		return nil
	}

	// Start watch request handler
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				req, err := stream.Recv()
				if err != nil {
					if !errors.Is(err, io.EOF) {
						log.ErrorCtx(ctx, "stream receive failed", zap.Error(err))
					}
					return
				}

				if errRcv := handleReceive(req); errRcv != nil {
					log.ErrorCtx(ctx, "handle receive failed", zap.Error(errRcv))
				}
			}
		}
	}()

	// Main update loop
	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return fmt.Errorf("watcher channel closed: client too slow to process updates")
			}

			watchMu.RLock()
			_, watching := watchedLogs[update.logId]
			watchMu.RUnlock()

			if !watching {
				// Not related update for us
				continue
			}

			if err := sendUpdate(update); err != nil {
				return fmt.Errorf("failed to send update: %w", err)
			}
		}
	}
}

//
// Helper from original code
//

func (c *lightConsensusRpc) broadcast(update *recordUpdate) {
	var slowWatchers []chan *recordUpdate

	c.watchers.Range(func(key, _ interface{}) bool {
		ch, ok := key.(chan *recordUpdate)
		if !ok {
			panic("invalid watcher type: expected chan *recordUpdate")
		}

		select {
		case ch <- update:
		default:
			slowWatchers = append(slowWatchers, ch)
		}

		return true
	})

	for _, ch := range slowWatchers {
		c.removeWatcher(ch)
		log.Warn("removed slow watcher due to full channel")
	}
}

func (c *lightConsensusRpc) addWatcher(ch chan *recordUpdate) {
	c.watchers.Store(ch, struct{}{})
}

func (c *lightConsensusRpc) removeWatcher(ch chan *recordUpdate) {
	c.watchers.Delete(ch)
	close(ch)
}

func (c *lightConsensusRpc) checkRead(ctx context.Context) (err error) {
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return consensuserr.ErrForbidden
	}

	nodeTypes := c.nodeconf.NodeTypes(peerId)
	for _, nodeType := range nodeTypes {
		switch nodeType {
		case nodeconf.NodeTypeCoordinator,
			nodeconf.NodeTypeTree,
			nodeconf.NodeTypeFile:
			return nil
		}
	}

	return consensuserr.ErrForbidden
}

func (c *lightConsensusRpc) checkWrite(ctx context.Context) (err error) {
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return consensuserr.ErrForbidden
	}

	nodeTypes := c.nodeconf.NodeTypes(peerId)
	for _, nodeType := range nodeTypes {
		switch nodeType {
		case nodeconf.NodeTypeCoordinator,
			nodeconf.NodeTypeTree:
			return nil
		}
	}

	return consensuserr.ErrForbidden
}

func recordsToProto(recs []consensus.Record) []*consensusproto.RawRecordWithId {
	res := make([]*consensusproto.RawRecordWithId, len(recs))
	for i, rec := range recs {
		res[i] = &consensusproto.RawRecordWithId{
			Payload: rec.Payload,
			Id:      rec.Id,
		}
	}

	return res
}
