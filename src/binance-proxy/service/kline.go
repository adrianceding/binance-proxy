package service

import (
	"binance-proxy/tool"
	"container/list"
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	spot "github.com/adshao/go-binance/v2"
	futures "github.com/adshao/go-binance/v2/futures"
)

type Kline = futures.Kline

type KlinesSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	si         symbolInterval
	klinesList *list.List
	klines     []Kline
	updateTime time.Time
}

func NewKlinesSrv(ctx context.Context, si symbolInterval) *KlinesSrv {
	s := &KlinesSrv{si: si}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.initCtx, s.initDone = context.WithCancel(context.Background())

	return s
}

func (s *KlinesSrv) connect() (doneC, stopC chan struct{}, err error) {
	if s.si.Class == SPOT {
		return spot.WsKlineServe(s.si.Symbol, s.si.Interval, s.wsHandlerSpot, s.errHandler)
	} else {
		return futures.WsKlineServe(s.si.Symbol, s.si.Interval, s.wsHandlerFutures, s.errHandler)
	}
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

func (s *KlinesSrv) wsHandlerSpot(event *spot.WsKlineEvent)       { s.wsHandler(event) }
func (s *KlinesSrv) wsHandlerFutures(event *futures.WsKlineEvent) { s.wsHandler(event) }

func (s *KlinesSrv) wsHandler(event interface{}) {
	if s.klinesList == nil {
		for d := tool.NewDelayIterator(); ; d.Delay() {
			var klines interface{}
			var err error
			if s.si.Class == SPOT {
				klines, err = spot.NewClient("", "").NewKlinesService().
					Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(1000).
					Do(context.Background())
			} else {
				klines, err = futures.NewClient("", "").NewKlinesService().
					Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(1000).
					Do(context.Background())
			}
			if err != nil {
				log.Errorf("%s.Get init klines error!Error:%s", s.si, err)
				continue
			}

			s.klinesList = list.New()

			if s.si.Class == SPOT {
				for _, v := range klines.([]*spot.Kline) {
					s.klinesList.PushBack(&Kline{
						OpenTime:                 v.OpenTime,
						Open:                     v.Open,
						High:                     v.High,
						Low:                      v.Low,
						Close:                    v.Close,
						Volume:                   v.Volume,
						CloseTime:                v.CloseTime,
						QuoteAssetVolume:         v.QuoteAssetVolume,
						TradeNum:                 v.TradeNum,
						TakerBuyBaseAssetVolume:  v.TakerBuyBaseAssetVolume,
						TakerBuyQuoteAssetVolume: v.TakerBuyQuoteAssetVolume,
					})
				}
			} else {
				for _, v := range klines.([]*futures.Kline) {
					s.klinesList.PushBack(&Kline{
						OpenTime:                 v.OpenTime,
						Open:                     v.Open,
						High:                     v.High,
						Low:                      v.Low,
						Close:                    v.Close,
						Volume:                   v.Volume,
						CloseTime:                v.CloseTime,
						QuoteAssetVolume:         v.QuoteAssetVolume,
						TradeNum:                 v.TradeNum,
						TakerBuyBaseAssetVolume:  v.TakerBuyBaseAssetVolume,
						TakerBuyQuoteAssetVolume: v.TakerBuyQuoteAssetVolume,
					})
				}
			}

			defer s.initDone()

			break
		}
	}

	// Merge kline
	var kline *Kline
	if s.si.Class == SPOT {
		k := event.(*spot.WsKlineEvent).Kline
		kline = &Kline{
			OpenTime:                 k.StartTime,
			Open:                     k.Open,
			High:                     k.High,
			Low:                      k.Low,
			Close:                    k.Close,
			Volume:                   k.Volume,
			CloseTime:                k.EndTime,
			QuoteAssetVolume:         k.QuoteVolume,
			TradeNum:                 k.TradeNum,
			TakerBuyBaseAssetVolume:  k.ActiveBuyVolume,
			TakerBuyQuoteAssetVolume: k.ActiveBuyQuoteVolume,
		}
	} else {
		k := event.(*futures.WsKlineEvent).Kline
		kline = &Kline{
			OpenTime:                 k.StartTime,
			Open:                     k.Open,
			High:                     k.High,
			Low:                      k.Low,
			Close:                    k.Close,
			Volume:                   k.Volume,
			CloseTime:                k.EndTime,
			QuoteAssetVolume:         k.QuoteVolume,
			TradeNum:                 k.TradeNum,
			TakerBuyBaseAssetVolume:  k.ActiveBuyVolume,
			TakerBuyQuoteAssetVolume: k.ActiveBuyQuoteVolume,
		}
	}

	if s.klinesList.Back().Value.(*Kline).OpenTime < kline.OpenTime {
		s.klinesList.PushBack(kline)
	} else if s.klinesList.Back().Value.(*Kline).OpenTime == kline.OpenTime {
		s.klinesList.Back().Value = kline
	}

	for s.klinesList.Len() > 1000 {
		s.klinesList.Remove(s.klinesList.Front())
	}

	klinesArr := make([]Kline, s.klinesList.Len())
	i := 0
	for elems := s.klinesList.Front(); elems != nil; elems.Next() {
		klinesArr[i] = *(elems.Value.(*Kline))
		i++
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	s.klines = klinesArr
	s.updateTime = time.Now()
}

func (s *KlinesSrv) GetKlines() []Kline {
	<-s.initCtx.Done()
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.klines
}
