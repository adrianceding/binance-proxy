package futures

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
	log "github.com/sirupsen/logrus"
)

func init() {
	// client.WebsocketKeepalive = true
}

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type FuturesSrv struct {
	mutex sync.RWMutex

	rawExchangeInfo []byte

	klinesSrv sync.Map // map[SymbolInterval]*FuturesKlines
	depthSrv  sync.Map // map[SymbolInterval]*FuturesDepth
}

func NewFuturesSrv() *FuturesSrv {
	s := &FuturesSrv{}
	s.start()
	return s
}

func (s *FuturesSrv) getExchangeInfo() ([]byte, error) {
	resp, err := http.Get("https://fapi.binance.com/fapi/v1/exchangeInfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (s *FuturesSrv) start() {
	go func() {
		for {
			for delay := 1; ; delay *= 2 {
				if delay > 60 {
					delay = 60
				}
				time.Sleep(time.Duration(delay-1) * time.Second)

				data, err := s.getExchangeInfo()
				if err != nil {
					log.Errorf("Futures exchangeInfo init error!Error:%s", err)
					continue
				}

				s.mutex.Lock()
				s.rawExchangeInfo = data
				s.mutex.Unlock()

				log.Debugf("Futures exchangeInfo update success!")

				break
			}

			time.Sleep(time.Second * 60)
		}
	}()
}

func (s *FuturesSrv) ExchangeInfo() []byte {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	r := make([]byte, len(s.rawExchangeInfo))
	copy(r, s.rawExchangeInfo)
	return r
}

func (s *FuturesSrv) Klines(symbol, interval string) []client.Kline {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	v, loaded := s.klinesSrv.LoadOrStore(si, NewFuturesKlinesSrv(si))
	srv := v.(*FuturesKlinesSrv)
	if loaded == false {
		srv.Start()
	}

	return srv.GetKlines()
}

func (s *FuturesSrv) Depth(symbol string) client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	v, loaded := s.klinesSrv.LoadOrStore(si, NewFuturesDepthSrv(si))
	srv := v.(*FuturesDepthSrv)
	if loaded == false {
		srv.Start()
	}

	return srv.GetDepth()
}
