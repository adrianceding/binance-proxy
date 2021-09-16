package spot

import (
	"sync"

	client "github.com/adshao/go-binance/v2"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Spot struct {
	mutex sync.RWMutex

	klinesSrv map[SymbolInterval]*SpotKlines
	depthSrv  map[SymbolInterval]*SpotDepth
}

func NewSpot() *Spot {
	t := &Spot{
		klinesSrv: make(map[SymbolInterval]*SpotKlines),
		depthSrv:  make(map[SymbolInterval]*SpotDepth),
	}

	return t
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
