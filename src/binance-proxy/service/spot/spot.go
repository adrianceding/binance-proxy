package spot

import (
	"sync"

	client "github.com/adshao/go-binance/v2"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type SpotSrv struct {
	mutex sync.RWMutex

	rawExchangeInfo []byte

	klinesSrv sync.Map // map[SymbolInterval]*SpotSrvKlines
	depthSrv  sync.Map // map[SymbolInterval]*SpotSrvDepth
}

func NewSpotSrv() *SpotSrv {
	t := &SpotSrv{}

	return t
}

func (s *SpotSrv) ExchangeInfo() []byte {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	r := make([]byte, len(s.rawExchangeInfo))
	copy(r, s.rawExchangeInfo)
	return r
}

func (s *SpotSrv) Klines(symbol, interval string) []client.Kline {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	v, loaded := s.klinesSrv.LoadOrStore(si, NewSpotKlinesSrv(si))
	srv := v.(*SpotKlinesSrv)
	if loaded == false {
		srv.Start()
	}

	return srv.GetKlines()
}

func (s *SpotSrv) Depth(symbol string) client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	v, loaded := s.klinesSrv.LoadOrStore(si, NewSpotDepthSrv(si))
	srv := v.(*SpotDepthSrv)
	if loaded == false {
		srv.Start()
	}

	return srv.GetDepth()
}
