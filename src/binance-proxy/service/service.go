package service

import (
	"context"
	"sync"
)

type Service struct {
	ctx    context.Context
	cancel context.CancelFunc

	class           Class
	exchangeInfoSrv *ExchangeInfoSrv
	klinesSrv       sync.Map // map[symbolInterval]*Klines
	depthSrv        sync.Map // map[symbolInterval]*Depth
}

func NewService(ctx context.Context, class Class) *Service {
	s := &Service{class: class}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.exchangeInfoSrv = NewExchangeInfoSrv(s.ctx, NewSymbolInterval(s.class, "", ""))
	s.exchangeInfoSrv.Start()

	return s
}

func (s *Service) ExchangeInfo() []byte {
	return s.exchangeInfoSrv.GetExchangeInfo()
}

func (s *Service) Klines(symbol, interval string) []*Kline {
	si := NewSymbolInterval(s.class, symbol, interval)
	srv, loaded := s.klinesSrv.LoadOrStore(*si, NewKlinesSrv(s.ctx, si))
	if loaded == false {
		srv.(*KlinesSrv).Start()
	}

	return srv.(*KlinesSrv).GetKlines()
}

func (s *Service) Depth(symbol string) *Depth {
	si := NewSymbolInterval(s.class, symbol, "")
	srv, loaded := s.klinesSrv.LoadOrStore(*si, NewDepthSrv(s.ctx, si))
	if loaded == false {
		srv.(*DepthSrv).Start()
	}

	return srv.(*DepthSrv).GetDepth()
}
