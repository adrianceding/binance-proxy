package service

import (
	"binance-proxy/tool"
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

	si         *symbolInterval
	depthShare *depthShare
}

type depthShare struct {
	rw sync.RWMutex

	LastUpdateID int64
	Time         int64
	TradeTime    int64
	Bids         [][2]string
	Asks         [][2]string
}

func (k *depthShare) delayDestrcut() {
	go func() {
		k.rw.Lock()
		defer k.rw.Unlock()
		depthSharePool.Put(k)
	}()
}

var depthSharePool = &sync.Pool{
	New: func() interface{} {
		return &depthShare{
			Bids: make([][2]string, 20),
			Asks: make([][2]string, 20),
		}
	},
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
			doneC, stopC, err := s.connect()
			if err != nil {
				log.Errorf("%s.Websocket depth connect error!Error:%s", s.si, err)
				continue
			}

			log.Debugf("%s.Websocket depth connect success!", s.si)
			select {
			case <-s.ctx.Done():
				stopC <- struct{}{}
				return
			case <-doneC:
			}

			log.Debugf("%s.Websocket depth disconnected!Reconnecting", s.si)
		}
	}()
}

func (s *DepthSrv) Stop() {
	s.cancel()
}

func (s *DepthSrv) connect() (doneC, stopC chan struct{}, err error) {
	if s.si.Class == SPOT {
		return spot.WsPartialDepthServe100Ms(
			s.si.Symbol, "20",
			func(event *spot.WsPartialDepthEvent) { s.wsHandler(event) },
			s.errHandler,
		)
	} else {
		return futures.WsPartialDepthServeWithRate(
			s.si.Symbol, 20, 100*time.Millisecond,
			func(event *futures.WsDepthEvent) { s.wsHandler(event) },
			s.errHandler,
		)
	}
}

func (s *DepthSrv) wsHandler(event interface{}) {
	if s.depthShare == nil {
		defer s.initDone()
	}

	depthShare := depthSharePool.Get().(*depthShare)

	if vi, ok := event.(*spot.WsPartialDepthEvent); ok {
		depthShare.LastUpdateID = vi.LastUpdateID
		depthShare.Time = time.Now().UnixNano() / 1e6
		depthShare.TradeTime = time.Now().UnixNano() / 1e6
		minLen := len(vi.Bids)
		if minLen > len(vi.Asks) {
			minLen = len(vi.Asks)
		}
		depthShare.Asks = depthShare.Asks[:minLen]
		depthShare.Bids = depthShare.Bids[:minLen]
		for i := 0; i < minLen; i++ {
			depthShare.Asks[i][0] = vi.Asks[i].Price
			depthShare.Asks[i][1] = vi.Asks[i].Quantity
			depthShare.Bids[i][0] = vi.Bids[i].Price
			depthShare.Bids[i][1] = vi.Bids[i].Quantity
		}
	} else if vi, ok := event.(*futures.WsDepthEvent); ok {
		depthShare.LastUpdateID = vi.LastUpdateID
		depthShare.Time = vi.Time
		depthShare.TradeTime = vi.TransactionTime
		minLen := len(vi.Bids)
		if minLen > len(vi.Asks) {
			minLen = len(vi.Asks)
		}
		depthShare.Asks = depthShare.Asks[:minLen]
		depthShare.Bids = depthShare.Bids[:minLen]
		for i := 0; i < minLen; i++ {
			depthShare.Asks[i][0] = vi.Asks[i].Price
			depthShare.Asks[i][1] = vi.Asks[i].Quantity
			depthShare.Bids[i][0] = vi.Bids[i].Price
			depthShare.Bids[i][1] = vi.Bids[i].Quantity
		}
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	if s.depthShare != nil {
		s.depthShare.delayDestrcut()
	}

	s.depthShare = depthShare
}

func (s *DepthSrv) errHandler(err error) {
	log.Errorf("%s.Depth websocket throw error!Error:%s", s.si, err)
}
