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
	srv := v.(*KlinesSrv)
	if loaded == false {
		srv.Start()
	}
	<-srv.initCtx.Done()

	srv.rw.RLock()
	klinesShare := srv.klinesShare
	klinesShare.rw.RLock()
	srv.rw.RUnlock()
	defer klinesShare.rw.RUnlock()

	startIndex := 0
	endIndex := len(klinesShare.klines) - 1
	if enableFakeKlines && time.Now().UnixNano()/1e6 > klinesShare.klines[endIndex][K_CloseTime].(int64) {
		endIndex++
	}

	if endIndex > limit {
		startIndex = endIndex - limit
	}

	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(klinesShare.klines[startIndex:endIndex])

	return
}

type resDepth struct {
	LastUpdateID int64       `json:"lastUpdateId"`
	Time         int64       `json:"E"`
	TradeTime    int64       `json:"T"`
	Bids         [][2]string `json:"bids"`
	Asks         [][2]string `json:"asks"`
}

var resDepthPool = &sync.Pool{
	New: func() interface{} {
		return &resDepth{
			Bids: make([][2]string, 20),
			Asks: make([][2]string, 20),
		}
	},
}

func (s *Service) Depth(symbol string, limit int, buf *bytes.Buffer) {
	si := NewSymbolInterval(s.class, symbol, "")
	v, loaded := s.depthSrv.LoadOrStore(*si, NewDepthSrv(s.ctx, si))
	srv := v.(*DepthSrv)
	if loaded == false {
		srv.Start()
	}
	<-srv.initCtx.Done()

	srv.rw.RLock()
	depthShare := srv.depthShare
	depthShare.rw.RLock()
	srv.rw.RUnlock()
	defer depthShare.rw.RUnlock()

	minLen := len(depthShare.Bids)
	if minLen > len(depthShare.Asks) {
		minLen = len(depthShare.Asks)
	}
	if minLen > limit {
		minLen = limit
	}

	resDepth := resDepthPool.Get().(*resDepth)
	defer resDepthPool.Put(resDepth)

	resDepth.LastUpdateID = depthShare.LastUpdateID
	resDepth.Time = depthShare.Time
	resDepth.TradeTime = depthShare.TradeTime
	resDepth.Asks = depthShare.Asks[:minLen]
	resDepth.Bids = depthShare.Bids[:minLen]

	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.Encode(resDepth)

	return
}
