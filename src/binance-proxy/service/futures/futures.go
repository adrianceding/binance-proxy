package futures

import (
	"log"
	"sync"
)

type SymbolInterval struct {
	Symbol   string
	Interval string
}

type Futures struct {
	lock chan struct{}
	once sync.Once

	maxIndex    uint64
	symbolIndex map[SymbolInterval]uint64
	waitingInit chan SymbolInterval

	klines []*FuturesKlines
	depth  []*FuturesDepth
	price  []*FuturesPrice
}

func NewFutures() *Futures {
	t := &Futures{
		lock:        make(chan struct{}, 1),
		symbolIndex: make(map[SymbolInterval]uint64),
		waitingInit: make(chan SymbolInterval, 1024),
		klines:      make([]*FuturesKlines, 0),
		depth:       make([]*FuturesDepth, 0),
		price:       make([]*FuturesPrice, 0),
	}
	go t.consumeWaitingInit()

	return t
}

func (s *Futures) consumeWaitingInit() {
	s.once.Do(func() {
		for si := range s.waitingInit {
			s.initSymbol(si)
		}
	})
}

func (s *Futures) initSymbol(si SymbolInterval) (err error) {
	defer func() { <-s.lock }()
	s.lock <- struct{}{}

	if _, ok := s.symbolIndex[si]; ok {
		log.Println("Futures ", si, " inited.Ignore")
		return
	}

	log.Println("Futures ", si, " initing.")

	s.klines = append(s.klines, nil)
	s.depth = append(s.depth, nil)
	s.price = append(s.price, nil)

	if s.klines[s.maxIndex], err = NewFutresKlines(si); err != nil {
		log.Println("Futures klines ", si, " init error.Err:", err)
		return
	}
	if s.depth[s.maxIndex], err = NewFutresDepth(si); err != nil {
		log.Println("Futures depth ", si, " init error.Err:", err)
		return
	}
	if s.price[s.maxIndex], err = NewFutresPrice(si); err != nil {
		log.Println("Futures price ", si, " init error.Err:", err)
		return
	}

	s.symbolIndex[si] = s.maxIndex

	s.maxIndex++

	log.Println("Futures ", si, " init success.")

	return
}

func (s *Futures) Klines(symbol, interval string) *FuturesKlines {
	defer func() { <-s.lock }()
	s.lock <- struct{}{}

	si := SymbolInterval{Symbol: symbol, Interval: interval}
	if index, ok := s.symbolIndex[si]; ok {
		return s.klines[index]
	}

	s.waitingInit <- si

	return nil
}

func (s *Futures) Depth(symbol string) *FuturesDepth {
	defer func() { <-s.lock }()
	s.lock <- struct{}{}

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	if index, ok := s.symbolIndex[si]; ok {
		return s.depth[index]
	}

	s.waitingInit <- si

	return nil
}

func (s *Futures) Price(symbol string) *FuturesPrice {
	defer func() { <-s.lock }()
	s.lock <- struct{}{}

	si := SymbolInterval{Symbol: symbol, Interval: ""}
	if index, ok := s.symbolIndex[si]; ok {
		return s.price[index]
	}

	s.waitingInit <- si

	return nil
}
