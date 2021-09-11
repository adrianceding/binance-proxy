package futures

import (
	"context"
	"log"
	"sync"

	client "github.com/adshao/go-binance/v2/futures"
)

const MAX_KLINE_NUM = 2000

type FuturesKlines struct {
	mutex sync.RWMutex

	waitInit chan struct{}

	SI     SymbolInterval
	klines []*client.Kline
}

func NewFutresKlines(si SymbolInterval) *FuturesKlines {
	k := &FuturesKlines{
		SI:       si,
		waitInit: make(chan struct{}, 0),
	}

	k.initWs()

	return k
}

func (s *FuturesKlines) GetKlines() []client.Kline {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	r := make([]client.Kline, 0)
	if s.isInited() == false {
		return r
	}

	for _, v := range s.klines {
		if v != nil {
			r = append(r, *v)
		}
	}

	return r
}

func (s *FuturesKlines) WsHandler(event *client.WsKlineEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	// Init klines
	if s.klines == nil && s.initApi() != nil {
		return
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
	defer s.mutex.Unlock()
	s.mutex.Lock()

	log.Printf("%s.Websocket throw error!Error:%s", s.SI, err)

	// TODO test errhandler action
	// s.klines = nil
	// for s.initWs() != nil {
	// 	time.Sleep(time.Second)
	// }
}

func (s *FuturesKlines) isInited() bool {
	// select {
	// case ret := <-do():
	// 	return ret, nil
	// case <-time.After(timeout):
	// 	return 0, errors.New("timeout")
	// }
	return true
}

func (s *FuturesKlines) initApi() error {
	klines, err := client.NewClient("", "").NewKlinesService().
		Symbol(s.SI.Symbol).Interval(s.SI.Interval).Limit(1500).
		Do(context.Background())
	if err != nil {
		log.Printf("%s.Get initialization klines error!Error:%s", s.SI, err)
		return err
	}

	s.klines = klines

	return nil
}

func (s *FuturesKlines) initWs() error {
	client.WebsocketKeepalive = true
	if _, _, err := client.WsKlineServe(s.SI.Symbol, s.SI.Interval, s.WsHandler, s.ErrHandler); err != nil {
		log.Printf("%s.Init websocket connection error!Error:%s", s.SI, err)
		return err
	}

	return nil
}
