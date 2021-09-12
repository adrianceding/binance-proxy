package futures

import (
	"context"
	"log"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
)

const MAX_KLINE_NUM = 2000

type FuturesKlines struct {
	mutex sync.RWMutex

	SI     SymbolInterval
	klines []*client.Kline
}

func NewFutresKlines(si SymbolInterval) *FuturesKlines {
	k := &FuturesKlines{SI: si}

	go func() {
		for {
			s.mutex.Lock()
			exitC := k.InitData()
			s.mutex.Unlock()

			select {
			case <-exitC:
			}
		}
	}()

	return k
}

func (s *FuturesKlines) InitData() chan strcut {
	client.WebsocketKeepalive = true

	doneC, _, err := client.WsKlineServe(s.SI.Symbol, s.SI.Interval, s.WsHandler, s.ErrHandler)
	if err != nil {
		log.Printf("%s.Init websocket connection error!Error:%s", s.SI, err)
		return false
	}

	for {
		if klines, err := client.NewClient("", "").NewKlinesService().
			Symbol(s.SI.Symbol).Interval(s.SI.Interval).Limit(1500).
			Do(context.Background()); err == nil {

			s.klines = klines
			break
		}

		log.Printf("%s.Get initialization klines error!Error:%s", s.SI, err)
		time.Sleep(time.Second)
	}

	s.mutex.Unlock()
}

func (s *FuturesKlines) GetKlines() []client.Kline {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	res := make([]client.Kline, 0)

	for _, v := range s.klines {
		if v != nil {
			res = append(res, *v)
		}
	}

	return res
}

func (s *FuturesKlines) WsHandler(event *client.WsKlineEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

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

	if len(s.klines) == 0 || s.klines[len(s.klines)-1].OpenTime < kline.OpenTime {
		s.klines = append(s.klines, kline)
	} else if s.klines[len(s.klines)-1].OpenTime == kline.OpenTime {
		s.klines[len(s.klines)-1] = kline
	}

	if len(s.klines) > MAX_KLINE_NUM {
		s.klines = s.klines[len(s.klines)-MAX_KLINE_NUM:]
	}
}

func (s *FuturesKlines) ErrHandler(err error) {
	log.Printf("%s.Websocket throw error!Error:%s", s.SI, err)
}
