package futures

import (
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Futures struct {
	mutex sync.RWMutex

	rawExchangeInfo []byte

	klinesSrv map[SymbolInterval]*FuturesKlines
	depthSrv  map[SymbolInterval]*FuturesDepth
}

func NewFutures() *Futures {
	s := &Futures{
		klinesSrv: make(map[SymbolInterval]*FuturesKlines),
		depthSrv:  make(map[SymbolInterval]*FuturesDepth),
	}
	s.start()
	return s
}

func (s *Futures) getExchangeInfo() ([]byte, error) {
	resp, err := http.Get("https://fapi.binance.com/fapi/v1/exchangeInfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (s *Futures) start() {
	go func() {
		for {
			for delay := 1; ; delay *= 2 {
				if delay > 60 {
					delay = 60
				}
				time.Sleep(time.Duration(delay-1) * time.Second)

				data, err := s.getExchangeInfo()
				if err != nil {
					log.Printf("Get futures exchange info error!Error:%s", err)
					continue
				}

				s.mutex.Lock()
				s.rawExchangeInfo = data
				s.mutex.Unlock()

				break
			}

			time.Sleep(time.Second * 60)
		}
	}()
}

func (s *Futures) ExchangeInfo() []byte {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	return s.rawExchangeInfo
}

func (s *Futures) Klines(symbol, interval string) []client.Kline {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	if _, ok := s.klinesSrv[si]; !ok {
		return nil
	}

	return s.klinesSrv[si].GetKlines()
}

func (s *Futures) Depth(symbol string) client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	if _, ok := s.depthSrv[si]; !ok {
		return client.DepthResponse{}
	}

	return s.depthSrv[si].GetDepth()
}
