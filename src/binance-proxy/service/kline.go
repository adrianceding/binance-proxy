package service

import (
	"binance-proxy/tool"
	"container/list"
	"context"
	"sync"

	log "github.com/sirupsen/logrus"

	spot "github.com/adshao/go-binance/v2"
	futures "github.com/adshao/go-binance/v2/futures"
)

const MAINTANCE_KLINES_LEN = 1000

const (
	K_OpenTime = iota
	K_Open
	K_High
	K_Low
	K_Close
	K_Volume
	K_CloseTime
	K_QuoteAssetVolume
	K_TradeNum
	K_TakerBuyBaseAssetVolume
	K_TakerBuyQuoteAssetVolume
	K_NoUse
)

// Save conversion
type Kline [12]interface {
	// OpenTime,
	// Open,
	// High,
	// Low,
	// Close,
	// Volume,
	// CloseTime,
	// QuoteAssetVolume,
	// TradeNum,
	// TakerBuyBaseAssetVolume,
	// TakerBuyQuoteAssetVolume,
	// "0",
}

type Klines []Kline
type KlinesShare struct {
	rw     sync.RWMutex
	klines Klines
}

var KlinePool = &sync.Pool{
	New: func() interface{} {
		return &Kline{}
	},
}

var KlinesPool = &sync.Pool{
	New: func() interface{} {
		return &KlinesShare{klines: make(Klines, MAINTANCE_KLINES_LEN+1)}
	},
}

type KlinesSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	si          *symbolInterval
	klinesList  *list.List
	klinesShare *KlinesShare

	FirstTradeID int64
	LastTradeID  int64
}

func NewKlinesSrv(ctx context.Context, si *symbolInterval) *KlinesSrv {
	s := &KlinesSrv{si: si}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.initCtx, s.initDone = context.WithCancel(context.Background())

	return s
}

func (s *KlinesSrv) Start() {
	go func() {
		for d := tool.NewDelayIterator(); ; d.Delay() {
			doneC, stopC, err := s.connect()
			if err != nil {
				log.Errorf("%s.Websocket klines connect error!Error:%s", s.si, err)
				continue
			}

			log.Debugf("%s.Websocket klines connect success!", s.si)
			select {
			case <-s.ctx.Done():
				stopC <- struct{}{}
				return
			case <-doneC:
			}

			log.Debugf("%s.Websocket klines disconnected!Reconnecting", s.si)
		}
	}()
}

func (s *KlinesSrv) Stop() {
	s.cancel()
}

func (s *KlinesSrv) errHandler(err error) {
	log.Errorf("%s.Klines websocket throw error!Error:%s", s.si, err)
}

func (s *KlinesSrv) connect() (doneC, stopC chan struct{}, err error) {
	if s.si.Class == SPOT {
		return spot.WsKlineServe(s.si.Symbol,
			s.si.Interval,
			func(event *spot.WsKlineEvent) { s.wsHandler(event) },
			s.errHandler,
		)
	} else {
		return futures.WsKlineServe(s.si.Symbol,
			s.si.Interval,
			func(event *futures.WsKlineEvent) { s.wsHandler(event) },
			s.errHandler,
		)
	}
}

func (s *KlinesSrv) initKlines() {
	for d := tool.NewDelayIterator(); ; d.Delay() {
		var klines interface{}
		var err error
		if s.si.Class == SPOT {
			SpotLimiter.WaitN(s.ctx, 1)
			klines, err = spot.NewClient("", "").NewKlinesService().
				Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(MAINTANCE_KLINES_LEN).
				Do(s.ctx)
		} else {
			FuturesLimiter.WaitN(s.ctx, 5)
			klines, err = futures.NewClient("", "").NewKlinesService().
				Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(MAINTANCE_KLINES_LEN).
				Do(s.ctx)
		}
		if err != nil {
			log.Errorf("%s.Get init klines error!Error:%s", s.si, err)
			continue
		}

		s.klinesList = list.New()

		if vi, ok := klines.([]*spot.Kline); ok {
			for _, v := range vi {
				k := KlinePool.Get().(*Kline)

				k[K_OpenTime] = v.OpenTime
				k[K_Open] = v.Open
				k[K_High] = v.High
				k[K_Low] = v.Low
				k[K_Close] = v.Close
				k[K_Volume] = v.Volume
				k[K_CloseTime] = v.CloseTime
				k[K_QuoteAssetVolume] = v.QuoteAssetVolume
				k[K_TradeNum] = v.TradeNum
				k[K_TakerBuyBaseAssetVolume] = v.TakerBuyBaseAssetVolume
				k[K_TakerBuyQuoteAssetVolume] = v.TakerBuyQuoteAssetVolume
				k[K_NoUse] = "0"

				s.klinesList.PushBack(k)
			}
		} else if vi, ok := klines.([]*futures.Kline); ok {
			for _, v := range vi {
				k := KlinePool.Get().(*Kline)

				k[K_OpenTime] = v.OpenTime
				k[K_Open] = v.Open
				k[K_High] = v.High
				k[K_Low] = v.Low
				k[K_Close] = v.Close
				k[K_Volume] = v.Volume
				k[K_CloseTime] = v.CloseTime
				k[K_QuoteAssetVolume] = v.QuoteAssetVolume
				k[K_TradeNum] = v.TradeNum
				k[K_TakerBuyBaseAssetVolume] = v.TakerBuyBaseAssetVolume
				k[K_TakerBuyQuoteAssetVolume] = v.TakerBuyQuoteAssetVolume
				k[K_NoUse] = "0"

				s.klinesList.PushBack(k)
			}
		}

		break
	}
}

func (s *KlinesSrv) wsHandler(event interface{}) {
	if s.klinesList == nil {
		s.initKlines()
		defer s.initDone()
	}

	// Merge kline
	k := KlinePool.Get().(*Kline)
	if vi, ok := event.(*spot.WsKlineEvent); ok {
		k[K_OpenTime] = vi.Kline.StartTime
		k[K_Open] = vi.Kline.Open
		k[K_High] = vi.Kline.High
		k[K_Low] = vi.Kline.Low
		k[K_Close] = vi.Kline.Close
		k[K_Volume] = vi.Kline.Volume
		k[K_CloseTime] = vi.Kline.EndTime
		k[K_QuoteAssetVolume] = vi.Kline.QuoteVolume
		k[K_TradeNum] = vi.Kline.TradeNum
		k[K_TakerBuyBaseAssetVolume] = vi.Kline.ActiveBuyVolume
		k[K_TakerBuyQuoteAssetVolume] = vi.Kline.ActiveBuyQuoteVolume
		k[K_NoUse] = "0"

		s.FirstTradeID = vi.Kline.FirstTradeID
		s.LastTradeID = vi.Kline.LastTradeID
	} else if vi, ok := event.(*futures.WsKlineEvent); ok {
		k[K_OpenTime] = vi.Kline.StartTime
		k[K_Open] = vi.Kline.Open
		k[K_High] = vi.Kline.High
		k[K_Low] = vi.Kline.Low
		k[K_Close] = vi.Kline.Close
		k[K_Volume] = vi.Kline.Volume
		k[K_CloseTime] = vi.Kline.EndTime
		k[K_QuoteAssetVolume] = vi.Kline.QuoteVolume
		k[K_TradeNum] = vi.Kline.TradeNum
		k[K_TakerBuyBaseAssetVolume] = vi.Kline.ActiveBuyVolume
		k[K_TakerBuyQuoteAssetVolume] = vi.Kline.ActiveBuyQuoteVolume
		k[K_NoUse] = "0"

		s.FirstTradeID = vi.Kline.FirstTradeID
		s.LastTradeID = vi.Kline.LastTradeID
	}

	if s.klinesList.Back().Value.(*Kline)[K_OpenTime].(int64) < k[K_OpenTime].(int64) {
		s.klinesList.PushBack(k)
	} else if s.klinesList.Back().Value.(*Kline)[K_OpenTime].(int64) == k[K_OpenTime].(int64) {
		KlinePool.Put(s.klinesList.Back().Value)
		s.klinesList.Back().Value = k
	}

	for s.klinesList.Len() > MAINTANCE_KLINES_LEN {
		KlinePool.Put(s.klinesList.Front().Value)
		s.klinesList.Remove(s.klinesList.Front())
	}

	//------------------------------------------------------------------------

	klinesShare := KlinesPool.Get().(*KlinesShare)

	klinesShare.klines = klinesShare.klines[:s.klinesList.Len()]
	i := 0
	for elems := s.klinesList.Front(); elems != nil; elems = elems.Next() {
		klinesShare.klines[i] = *(elems.Value.(*Kline))
		i++
	}
	// Add fake kline
	lastK := klinesShare.klines[len(klinesShare.klines)-1]
	closeTime := lastK[K_CloseTime].(int64)
	openTime := lastK[K_OpenTime].(int64)

	klinesShare.klines = append(klinesShare.klines, Kline{
		K_OpenTime:                 closeTime + 1,
		K_Open:                     lastK[K_Close],
		K_High:                     lastK[K_Close],
		K_Low:                      lastK[K_Close],
		K_Close:                    lastK[K_Close],
		K_Volume:                   "0.0",
		K_CloseTime:                closeTime + 1 + (closeTime - openTime),
		K_QuoteAssetVolume:         "0.0",
		K_TradeNum:                 0,
		K_TakerBuyBaseAssetVolume:  "0.0",
		K_TakerBuyQuoteAssetVolume: "0.0",
		K_NoUse:                    "0",
	})

	//------------------------------------------------------------------------

	s.rw.Lock()
	defer s.rw.Unlock()

	if s.klinesShare != nil {
		go func(k *KlinesShare) {
			k.rw.Lock()
			defer k.rw.Unlock()
			KlinesPool.Put(k)
		}(s.klinesShare)
	}

	s.klinesShare = klinesShare
}
