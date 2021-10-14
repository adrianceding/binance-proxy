package service

import (
	"binance-proxy/internal/tool"
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	spot "github.com/adshao/go-binance/v2"
	futures "github.com/adshao/go-binance/v2/futures"
)

type DepthSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	si    *symbolInterval
	depth *Depth
}

type Depth struct {
	LastUpdateID int64
	Time         int64
	TradeTime    int64
	Bids         []futures.Bid
	Asks         []futures.Ask
}

func NewDepthSrv(ctx context.Context, si *symbolInterval) *DepthSrv {
	s := &DepthSrv{si: si}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.initCtx, s.initDone = context.WithCancel(context.Background())

	return s
}

func (s *DepthSrv) Start() {
	go func() {
		for d := tool.NewDelayIterator(); ; d.Delay() {
			s.rw.Lock()
			s.depth = nil
			s.rw.Unlock()

			doneC, stopC, err := s.connect()
			if err != nil {
				log.Errorf("%s %s depth websocket connection error: %s.", s.si.Class, s.si.Symbol, err)
				continue
			}

			log.Debugf("%s %s depth websocket connected.", s.si.Class, s.si.Symbol)
			select {
			case <-s.ctx.Done():
				stopC <- struct{}{}
				return
			case <-doneC:
			}

			log.Warnf("%s %s depth websocket disconnected, trying to reconnect.", s.si.Class, s.si.Symbol)
		}
	}()
}

func (s *DepthSrv) Stop() {
	s.cancel()
}

func (s *DepthSrv) connect() (doneC, stopC chan struct{}, err error) {
	if s.si.Class == SPOT {
		return spot.WsPartialDepthServe100Ms(s.si.Symbol, "20", s.wsHandler, s.errHandler)
	} else {
		return futures.WsPartialDepthServeWithRate(s.si.Symbol, 20, 100*time.Millisecond, s.wsHandlerFutures, s.errHandler)
	}
}

func (s *DepthSrv) GetDepth() *Depth {
	<-s.initCtx.Done()
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.depth
}

func (s *DepthSrv) wsHandlerFutures(event *futures.WsDepthEvent) {
	s.rw.Lock()
	defer s.rw.Unlock()

	if s.depth == nil {
		defer s.initDone()
	}

	s.depth = &Depth{
		LastUpdateID: event.LastUpdateID,
		Time:         event.Time,
		TradeTime:    event.TransactionTime,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
	log.Tracef("%s %s depth websocket message received", s.si.Class, s.si.Symbol)
}

func (s *DepthSrv) wsHandler(event *spot.WsPartialDepthEvent) {
	s.rw.Lock()
	defer s.rw.Unlock()

	if s.depth == nil {
		defer s.initDone()
	}

	s.depth = &Depth{
		LastUpdateID: event.LastUpdateID,
		Time:         time.Now().UnixNano() / 1e6,
		TradeTime:    time.Now().UnixNano() / 1e6,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
	log.Tracef("%s %s depth websocket message received", s.si.Class, s.si.Symbol)

}

func (s *DepthSrv) errHandler(err error) {
	log.Errorf("%s %s depth websocket connection error: %s.", s.si.Class, s.si.Symbol, err)

}
