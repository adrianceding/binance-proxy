package service

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"
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

func (s *Service) Klines(symbol, interval string, enableFakeKlines bool, limit int, buf *bytes.Buffer) {
	si := NewSymbolInterval(s.class, symbol, interval)
	v, loaded := s.klinesSrv.LoadOrStore(*si, NewKlinesSrv(s.ctx, si))
	ksrv := v.(*KlinesSrv)
	if loaded == false {
		ksrv.Start()
	}
	<-ksrv.initCtx.Done()

	ksrv.rw.RLock()
	ksrv.klinesShare.rw.RLock()
	ksrv.rw.RUnlock()
	defer ksrv.klinesShare.rw.RUnlock()

	startIndex := 0
	endIndex := len(ksrv.klinesShare.klines) - 1
	if enableFakeKlines && time.Now().UnixNano()/1e6 > ksrv.klinesShare.klines[endIndex][K_CloseTime].(int64) {
		endIndex++
	}

	if endIndex > limit {
		startIndex = endIndex - limit
	}

	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(ksrv.klinesShare.klines[startIndex:endIndex])

	return
}

func (s *Service) Depth(symbol string) *Depth {
	si := NewSymbolInterval(s.class, symbol, "")
	srv, loaded := s.klinesSrv.LoadOrStore(*si, NewDepthSrv(s.ctx, si))
	if loaded == false {
		srv.(*DepthSrv).Start()
	}

	return srv.(*DepthSrv).GetDepth()
}
