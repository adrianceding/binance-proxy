package futures

import (
	"container/list"
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	client "github.com/adshao/go-binance/v2/futures"
)

type FuturesKlinesSrv struct {
	mutex sync.RWMutex

	onceStart  sync.Once
	onceInited sync.Once
	onceStop   sync.Once

	inited chan struct{}
	stopC  chan struct{}
	errorC chan struct{}

	si         SymbolInterval
	klines     *list.List
	updateTime time.Time
}

func NewFuturesKlinesSrv(si SymbolInterval) *FuturesKlinesSrv {
	return &FuturesKlinesSrv{
		si:     si,
		stopC:  make(chan struct{}, 1),
		errorC: make(chan struct{}, 1),
		inited: make(chan struct{}),
	}
}

func (s *FuturesKlinesSrv) Start() {
	go func() {
		for delay := 1; ; delay *= 2 {
			s.mutex.Lock()
			s.klines = nil
			s.mutex.Unlock()

			if delay > 60 {
				delay = 60
			}
			time.Sleep(time.Second * time.Duration(delay-1))

			doneC, stopC, err := client.WsKlineServe(s.si.Symbol, s.si.Interval, s.wsHandler, s.errHandler)
			if err != nil {
				log.Errorf("%s.Futures websocket klines connect error!Error:%s", s.si, err)
				continue
			}

			delay = 1
			select {
			case <-s.stopC:
				log.Debugf("%s.Stop enter", s.si)
				stopC <- struct{}{}
				log.Debugf("%s.Stop leave", s.si)
				return
			case <-doneC:
				log.Debugf("%s.DoneDoneDoneDoneDoneDoneDoneDoneDoneDoneDoneDoneDoneDone", s.si)
			case <-s.errorC:
			}

			log.Debugf("%s.Futures websocket klines disconnected!Reconnecting", s.si)
		}
	}()
}

func (s *FuturesKlinesSrv) Stop() {
	s.onceStop.Do(func() {
		s.stopC <- struct{}{}
	})
}

func (s *FuturesKlinesSrv) wsHandler(event *client.WsKlineEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	if s.klines == nil {
		for delay := 1; ; delay *= 2 {
			if delay > 60 {
				delay = 60
			}
			time.Sleep(time.Second * time.Duration(delay-1))

			klines, err := client.NewClient("", "").NewKlinesService().
				Symbol(s.si.Symbol).Interval(s.si.Interval).Limit(1000).
				Do(context.Background())
			if err != nil {
				log.Errorf("%s.Get initialization api klines error!Error:%s", s.si, err)
				continue
			}

			s.klines = list.New()
			for _, v := range klines {
				s.klines.PushBack(v)
			}
			break
		}
	}

	// Merge kline
	kline := &client.Kline{
		OpenTime:                 event.Kline.StartTime,
		Open:                     event.Kline.Open,
		High:                     event.Kline.High,
		Low:                      event.Kline.Low,
		Close:                    event.Kline.Close,
		Volume:                   event.Kline.Volume,
		CloseTime:                event.Kline.EndTime,
		QuoteAssetVolume:         event.Kline.QuoteVolume,
		TradeNum:                 event.Kline.TradeNum,
		TakerBuyBaseAssetVolume:  event.Kline.ActiveBuyVolume,
		TakerBuyQuoteAssetVolume: event.Kline.ActiveBuyQuoteVolume,
	}

	if s.klines.Back().Value.(*client.Kline).OpenTime < kline.OpenTime {
		s.klines.PushBack(kline)
	} else if s.klines.Back().Value.(*client.Kline).OpenTime == kline.OpenTime {
		s.klines.Back().Value = kline
	}

	for s.klines.Len() > 1000 {
		s.klines.Remove(s.klines.Front())
	}
	s.updateTime = time.Now()

	s.onceInited.Do(func() {
		log.Debugf("%s.Futures klines init success!", s.si)
		close(s.inited)
	})
}

func (s *FuturesKlinesSrv) errHandler(err error) {
	log.Errorf("%s.Futures klines websocket throw error!Error:%s", s.si, err)
	s.errorC <- struct{}{}
}

func (s *FuturesKlinesSrv) GetKlines() []client.Kline {
	<-s.inited

	defer s.mutex.RUnlock()
	s.mutex.RLock()

	// if time.Now().Sub(s.updateTime).Seconds() > 5 {
	// 	return nil
	// }
	if s.klines == nil {
		return nil
	}

	res := make([]client.Kline, s.klines.Len())
	elems := s.klines.Front()
	for i := 0; ; i++ {
		if elems == nil {
			break
		} else {
			res[i] = *(elems.Value.(*client.Kline))
		}
		elems = elems.Next()
	}

	return res
}
