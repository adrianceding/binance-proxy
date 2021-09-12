package futures

import (
	"sync"

	client "github.com/adshao/go-binance/v2/futures"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Futures struct {
	mutex sync.RWMutex

	klinesSrv map[SymbolInterval]*FuturesKlines
	depthSrv  map[SymbolInterval]*FuturesDepth
}

func NewFutures() *Futures {
	t := &Futures{
		klinesSrv: make(map[SymbolInterval]*FuturesKlines),
		depthSrv:  make(map[SymbolInterval]*FuturesDepth),
	}

	return t
}

func (s *Futures) Klines(symbol, interval string) []client.Kline {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	if _, ok := s.klinesSrv[si]; !ok {
		s.klinesSrv[si] = NewFutresKlines(si)
	}

	return s.klinesSrv[si].GetKlines()
}

func (s *Futures) Depth(symbol string) client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	if _, ok := s.depthSrv[si]; !ok {
		s.depthSrv[si] = NewFutresDepth(si)
	}

	return s.depthSrv[si].GetDepth()
}
