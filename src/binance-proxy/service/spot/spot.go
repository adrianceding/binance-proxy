package spot

import (
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Spot struct {
	mutex sync.RWMutex

	rawExchangeInfo []byte

	klinesSrv map[SymbolInterval]*SpotKlines
	depthSrv  map[SymbolInterval]*SpotDepth
}

func NewSpot() *Spot {
	t := &Spot{
		klinesSrv: make(map[SymbolInterval]*SpotKlines),
		depthSrv:  make(map[SymbolInterval]*SpotDepth),
	}

	t.start()

	return t
}

func (s *Spot) getExchangeInfo() ([]byte, error) {
	resp, err := http.Get("https://api.binance.com/api/v3/exchangeInfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (s *Spot) ExchangeInfo() []byte {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	return s.rawExchangeInfo
}

func (s *Spot) start() {
	go func() {
		for {
			for delay := 1; ; delay *= 2 {
				if delay > 60 {
					delay = 60
				}
				time.Sleep(time.Duration(delay-1) * time.Second)

				data, err := s.getExchangeInfo()
				if err != nil {
					log.Printf("Get spot exchange info error!Error:%s", err)
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

func (s *Spot) Klines(symbol, interval string) []client.Kline {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	if _, ok := s.klinesSrv[si]; !ok {
		return nil
	}

	return s.klinesSrv[si].GetKlines()
}

func (s *Spot) Depth(symbol string) client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	if _, ok := s.depthSrv[si]; !ok {
		return client.DepthResponse{}
	}

	return s.depthSrv[si].GetDepth()
}
