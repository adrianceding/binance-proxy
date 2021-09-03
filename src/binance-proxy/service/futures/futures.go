package futures

import (
	"errors"
	"sync"
	"sync/atomic"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Futures struct {
	maxIndex    uint64
	symbolIndex sync.Map // map[SymbolInterval]uint64
	waitingInit chan SymbolInterval

	klines []*FuturesKlines
	depth  []*FuturesDepth
	price  []*FuturesPrice
}

func NewFutures() *Futures {
	t := &Futures{
		waitingInit: make(chan SymbolInterval, 1024),
	}
	go t.consumeWaitingInit()

	return t
}

func (s *Futures) consumeWaitingInit() {
	for si := range s.waitingInit {
		if _, ok := s.symbolIndex.Load(si); ok {
			continue
		}

		index := atomic.AddUint64(&s.maxIndex, 1)

		if v, err := NewFutresKlines(si); err != nil {
			return
		} else {
			s.klines[index] = v
		}
		if v, err := NewFutresDepth(si); err != nil {
			return
		} else {
			s.depth[index] = v
		}
		if v, err := NewFutresPrice(si); err != nil {
			return
		} else {
			s.price[index] = v
		}

		s.symbolIndex.Store(si, index)
	}
}

func (s *Futures) getIndex(si SymbolInterval) (uint64, error) {
	if v, ok := s.symbolIndex.Load(si); ok {
		return v.(uint64), nil
	}

	s.waitingInit <- si

	return 0, errors.New("symbol need init")
}

func (s *Futures) Klines(symbol, interval string) *FuturesKlines {
	if index, err := s.getIndex(SymbolInterval{Symbol: symbol, Interval: interval}); err == nil {
		return s.klines[index]
	}

	return nil
}

func (s *Futures) Depth(symbol string) *FuturesDepth {
	if index, err := s.getIndex(SymbolInterval{Symbol: symbol, Interval: ""}); err == nil {
		return s.depth[index]
	}

	return nil
}

func (s *Futures) Price(symbol string) *FuturesPrice {
	if index, err := s.getIndex(SymbolInterval{Symbol: symbol, Interval: ""}); err == nil {
		return s.price[index]
	}

	return nil
}
