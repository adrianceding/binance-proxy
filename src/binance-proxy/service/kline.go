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
type Kline [12]interface{}

// {
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
// }

type KlinesSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	si         *symbolInterval
	klinesList *list.List
	klinesArr  []*Kline

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

func (s *KlinesSrv) wsHandler(event interface{}) {
	if s.klinesList == nil {
		for d := tool.NewDelayIterator(); ; d.Delay() {
			var klines interface{}
			var err error
			if s.si.Class == SPOT {
				SpotLimiter.WaitN(s.ctx, 1)
				klines, err = spot.NewClient("", "").NewKlinesService().
					Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(1000).
					Do(s.ctx)
			} else {
				FuturesLimiter.WaitN(s.ctx, 5)
				klines, err = futures.NewClient("", "").NewKlinesService().
					Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(1000).
					Do(s.ctx)
			}
			if err != nil {
				log.Errorf("%s.Get init klines error!Error:%s", s.si, err)
				continue
			}

			s.klinesList = list.New()

			if vi, ok := klines.([]*spot.Kline); ok {
				for _, v := range vi {
					t := &Kline{
						K_OpenTime:                 v.OpenTime,
						K_Open:                     v.Open,
						K_High:                     v.High,
						K_Low:                      v.Low,
						K_Close:                    v.Close,
						K_Volume:                   v.Volume,
						K_CloseTime:                v.CloseTime,
						K_QuoteAssetVolume:         v.QuoteAssetVolume,
						K_TradeNum:                 v.TradeNum,
						K_TakerBuyBaseAssetVolume:  v.TakerBuyBaseAssetVolume,
						K_TakerBuyQuoteAssetVolume: v.TakerBuyQuoteAssetVolume,
						K_NoUse:                    "0",
					}

					s.klinesList.PushBack(t)
				}
			} else if vi, ok := klines.([]*futures.Kline); ok {
				for _, v := range vi {
					t := &Kline{
						K_OpenTime:                 v.OpenTime,
						K_Open:                     v.Open,
						K_High:                     v.High,
						K_Low:                      v.Low,
						K_Close:                    v.Close,
						K_Volume:                   v.Volume,
						K_CloseTime:                v.CloseTime,
						K_QuoteAssetVolume:         v.QuoteAssetVolume,
						K_TradeNum:                 v.TradeNum,
						K_TakerBuyBaseAssetVolume:  v.TakerBuyBaseAssetVolume,
						K_TakerBuyQuoteAssetVolume: v.TakerBuyQuoteAssetVolume,
						K_NoUse:                    "0",
					}

					s.klinesList.PushBack(t)
				}
			}

			defer s.initDone()

			break
		}
	}

	// Merge kline
	var k *Kline
	if vi, ok := event.(*spot.WsKlineEvent); ok {
		k = &Kline{
			K_OpenTime:                 vi.Kline.StartTime,
			K_Open:                     vi.Kline.Open,
			K_High:                     vi.Kline.High,
			K_Low:                      vi.Kline.Low,
			K_Close:                    vi.Kline.Close,
			K_Volume:                   vi.Kline.Volume,
			K_CloseTime:                vi.Kline.EndTime,
			K_QuoteAssetVolume:         vi.Kline.QuoteVolume,
			K_TradeNum:                 vi.Kline.TradeNum,
			K_TakerBuyBaseAssetVolume:  vi.Kline.ActiveBuyVolume,
			K_TakerBuyQuoteAssetVolume: vi.Kline.ActiveBuyQuoteVolume,
			K_NoUse:                    "0",
		}
		s.FirstTradeID = vi.Kline.FirstTradeID
		s.LastTradeID = vi.Kline.LastTradeID
	} else if vi, ok := event.(*futures.WsKlineEvent); ok {
		k = &Kline{
			K_OpenTime:                 vi.Kline.StartTime,
			K_Open:                     vi.Kline.Open,
			K_High:                     vi.Kline.High,
			K_Low:                      vi.Kline.Low,
			K_Close:                    vi.Kline.Close,
			K_Volume:                   vi.Kline.Volume,
			K_CloseTime:                vi.Kline.EndTime,
			K_QuoteAssetVolume:         vi.Kline.QuoteVolume,
			K_TradeNum:                 vi.Kline.TradeNum,
			K_TakerBuyBaseAssetVolume:  vi.Kline.ActiveBuyVolume,
			K_TakerBuyQuoteAssetVolume: vi.Kline.ActiveBuyQuoteVolume,
			K_NoUse:                    "0",
		}
		s.FirstTradeID = vi.Kline.FirstTradeID
		s.LastTradeID = vi.Kline.LastTradeID
	}

	if s.klinesList.Back().Value.(*Kline)[K_OpenTime].(int64) < k[K_OpenTime].(int64) {
		s.klinesList.PushBack(k)
	} else if s.klinesList.Back().Value.(*Kline)[K_OpenTime].(int64) == k[K_OpenTime].(int64) {
		s.klinesList.Back().Value = k
	}

	for s.klinesList.Len() > 1000 {
		s.klinesList.Remove(s.klinesList.Front())
	}

	klinesArr := make([]*Kline, s.klinesList.Len())
	i := 0
	for elems := s.klinesList.Front(); elems != nil; elems = elems.Next() {
		klinesArr[i] = elems.Value.(*Kline)
		i++
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	s.klinesArr = klinesArr
}

func (s *KlinesSrv) GetKlines() []*Kline {
	<-s.initCtx.Done()
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.klinesArr
}
