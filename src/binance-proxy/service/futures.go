package service

import (
	"sync"
)

func NewFutures() *Futures {
	return &Futures{}
}

type Futures struct {
	listeningSymbol sync.Map
}

type FuturesKline struct {
}

type FuturesDepth struct {
}

type FuturesTickr struct {
}

// Klines map[string][]FuturesKline
// 	Depth  FuturesDepth
// 	Ticker FuturesTickr

func (s *Futures) klines(symbol, interval string) {

}

func (s *Futures) klines(symbol, interval string) {

}

func (s *Futures) depth(symbol string, limit int) {

}

func (s *Futures) price(symbol string) {

}
