package futures

import (
	"log"
	"sync"

	client "github.com/adshao/go-binance/v2/futures"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Futures struct {
	mutex sync.RWMutex

	maxIndex       uint64
	symbolIndex    map[SymbolInterval]uint64
	needInitSymbol chan SymbolInterval

	klinesSrv []*FuturesKlines
	depthSrv  []*FuturesDepth
}

func NewFutures() *Futures {
	t := &Futures{
		symbolIndex:    make(map[SymbolInterval]uint64),
		needInitSymbol: make(chan SymbolInterval, 1024),

		klinesSrv: make([]*FuturesKlines, 0),
		depthSrv:  make([]*FuturesDepth, 0),
	}
	go t.consume()

	return t
}

func (s *Futures) consume() {
	for si := range s.needInitSymbol {
		s.initSymbol(si)
	}
}

func (s *Futures) initSymbol(si SymbolInterval) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	if _, ok := s.symbolIndex[si]; ok {
		log.Println("Futures ", si, " inited.Ignore")
		return
	}

	log.Println("Futures ", si, " initing.")

	s.klinesSrv = append(s.klinesSrv, nil)
	s.depthSrv = append(s.depthSrv, nil)

	var err error
	if s.klinesSrv[s.maxIndex], err = NewFutresKlines(si); err != nil {
		log.Println("Futures klines ", si, " init error.Err:", err)
		return
	}
	if s.depthSrv[s.maxIndex], err = NewFutresDepth(si); err != nil {
		log.Println("Futures depth ", si, " init error.Err:", err)
		return
	}

	s.symbolIndex[si] = s.maxIndex

	s.maxIndex++

	log.Println("Futures ", si, " init success.")

	return
}

func (s *Futures) Klines(symbol, interval string) []*client.Kline {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	if index, ok := s.symbolIndex[si]; ok {
		return s.klinesSrv[index].Klines
	}
	s.needInitSymbol <- si

	return nil
}

func (s *Futures) Depth(symbol string) *client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	if index, ok := s.symbolIndex[si]; ok {
		return s.depthSrv[index].Depth
	}
	s.needInitSymbol <- si

	return nil
}
