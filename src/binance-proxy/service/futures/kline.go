package futures

import (
	"context"
	"log"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
)

type FuturesKlines struct {
	mutex sync.RWMutex

	SI     SymbolInterval
	Klines []*client.Kline
}

func NewFutresKlines(si SymbolInterval) (*FuturesKlines, error) {
	k := &FuturesKlines{
		SI: si,
	}
	if err := k.initWs(); err != nil {
		return nil, err
	}

	return k, nil
}

func (s *FuturesKlines) initWs() error {
	client.WebsocketKeepalive = true
	if _, _, err := client.WsKlineServe(s.SI.Symbol, s.SI.Interval, s.wsHandler, s.errHandler); err != nil {
		log.Println("kline init ws ", s.SI, " error:", err)
		return err
	}
	log.Println("kline init ws ", s.SI, " success.")

	return nil
}

func (s *FuturesKlines) wsHandler(event *client.WsKlineEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	if len(s.Klines) == 0 {
		var err error
		s.Klines, err = client.NewClient("", "").NewKlinesService().Symbol(s.SI.Symbol).
			Interval(s.SI.Interval).Limit(1000).Do(context.Background())
		if err != nil {
			log.Println("kline ws get klines ", s.SI, ".error:", err)
			return
		}
	}

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
	if len(s.Klines) == 0 {
		s.Klines = append(s.Klines, kline)
	}

	if s.Klines[len(s.Klines)-1].OpenTime == kline.OpenTime {
		s.Klines[len(s.Klines)-1] = kline
	} else if s.Klines[len(s.Klines)-1].OpenTime < kline.OpenTime {
		s.Klines = append(s.Klines, kline)
	}

	if len(s.Klines) > 2000 {
		s.Klines = s.Klines[len(s.Klines)-2000:]
	}
}

func (s *FuturesKlines) errHandler(err error) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	log.Println("kline ws err handler ", s.SI, ".error:", err)

	s.Klines = nil

	for {
		if s.initWs() == nil {
			break
		}
		time.Sleep(time.Second)
	}
}
