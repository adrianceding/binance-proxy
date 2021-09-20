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

type Kline struct {
	OpenTime                 int64
	Open                     string
	High                     string
	Low                      string
	Close                    string
	Volume                   string
	CloseTime                int64
	QuoteAssetVolume         string
	TradeNum                 int64
	TakerBuyBaseAssetVolume  string
	TakerBuyQuoteAssetVolume string

	IsFinal      bool
	FirstTradeID int64
	LastTradeID  int64
}

type KlinesSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	si         symbolInterval
	klinesList *list.List
	klinesArr  []Kline
	updateTime time.Time
}

func NewKlinesSrv(ctx context.Context, si symbolInterval) *KlinesSrv {
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

			if vi, ok := klines.([]*spot.Kline); ok {
				for k, v := range vi {
					t := &Kline{
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
						IsFinal:                  k+1 < len(vi),
					}

					s.klinesList.PushBack(t)
				}
			} else if vi, ok := klines.([]*futures.Kline); ok {
				for k, v := range vi {
					t := &Kline{
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
						IsFinal:                  k+1 < len(vi),
					}

					s.klinesList.PushBack(t)
				}
			}

			defer s.initDone()

			break
		}
	}

	// Merge kline
	var kline *Kline
	if vi, ok := event.(*spot.WsKlineEvent); ok {
		kline = &Kline{
			OpenTime:                 vi.Kline.StartTime,
			Open:                     vi.Kline.Open,
			High:                     vi.Kline.High,
			Low:                      vi.Kline.Low,
			Close:                    vi.Kline.Close,
			Volume:                   vi.Kline.Volume,
			CloseTime:                vi.Kline.EndTime,
			QuoteAssetVolume:         vi.Kline.QuoteVolume,
			TradeNum:                 vi.Kline.TradeNum,
			TakerBuyBaseAssetVolume:  vi.Kline.ActiveBuyVolume,
			TakerBuyQuoteAssetVolume: vi.Kline.ActiveBuyQuoteVolume,
			IsFinal:                  vi.Kline.IsFinal,
			FirstTradeID:             vi.Kline.FirstTradeID,
			LastTradeID:              vi.Kline.LastTradeID,
		}
	} else if vi, ok := event.(*futures.WsKlineEvent); ok {
		kline = &Kline{
			OpenTime:                 vi.Kline.StartTime,
			Open:                     vi.Kline.Open,
			High:                     vi.Kline.High,
			Low:                      vi.Kline.Low,
			Close:                    vi.Kline.Close,
			Volume:                   vi.Kline.Volume,
			CloseTime:                vi.Kline.EndTime,
			QuoteAssetVolume:         vi.Kline.QuoteVolume,
			TradeNum:                 vi.Kline.TradeNum,
			TakerBuyBaseAssetVolume:  vi.Kline.ActiveBuyVolume,
			TakerBuyQuoteAssetVolume: vi.Kline.ActiveBuyQuoteVolume,
			IsFinal:                  vi.Kline.IsFinal,
			FirstTradeID:             vi.Kline.FirstTradeID,
			LastTradeID:              vi.Kline.LastTradeID,
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
	for elems := s.klinesList.Front(); elems != nil; elems = elems.Next() {
		klinesArr[i] = *(elems.Value.(*Kline))
		i++
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	s.klinesArr = klinesArr
	s.updateTime = time.Now()
}

func (s *KlinesSrv) GetKlines() []Kline {
	<-s.initCtx.Done()
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.klinesArr
}
