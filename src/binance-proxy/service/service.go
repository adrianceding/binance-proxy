package service

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Service struct {
	ctx    context.Context
	cancel context.CancelFunc

	class           Class
	exchangeInfoSrv *ExchangeInfoSrv
	klinesSrv       sync.Map // map[symbolInterval]*Klines
	depthSrv        sync.Map // map[symbolInterval]*Depth

	lastGetKlines sync.Map // map[symbolInterval]time.Time
	lastGetDepth  sync.Map // map[symbolInterval]time.Time
}

func NewService(ctx context.Context, class Class) *Service {
	s := &Service{class: class}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.exchangeInfoSrv = NewExchangeInfoSrv(s.ctx, NewSymbolInterval(s.class, "", ""))
	s.exchangeInfoSrv.Start()

	go func() {
		for {
			t := time.NewTimer(time.Second)
			for {
				t.Reset(time.Second)
				select {
				case <-s.ctx.Done():
					t.Stop()
					return
				case <-t.C:
				}

				s.autoRemoveExpired()
			}
		}
	}()

	return s
}

func (s *Service) ExchangeInfo() []byte {
	return s.exchangeInfoSrv.GetExchangeInfo()
}

func (s *Service) autoRemoveExpired() {
	s.klinesSrv.Range(func(k, v interface{}) bool {
		si := k.(symbolInterval)
		srv := v.(*KlinesSrv)

		if t, ok := s.lastGetKlines.Load(si); ok {
			if time.Now().Sub(t.(time.Time)) > 2*INTERVAL_2_DURATION[si.Interval] {
				log.Debugf("%s.Kline srv expired!Removed", si)
				s.lastGetKlines.Delete(si)

				s.klinesSrv.Delete(si)
				srv.Stop()
			}
		} else {
			s.lastGetKlines.Store(si, time.Now())
		}

		return true
	})
	s.depthSrv.Range(func(k, v interface{}) bool {
		si := k.(symbolInterval)
		srv := v.(*DepthSrv)

		if t, ok := s.lastGetDepth.Load(si); ok {
			if time.Now().Sub(t.(time.Time)) > 2*time.Minute {
				log.Debugf("%s.Depth srv expired!Removed", si)
				s.lastGetDepth.Delete(si)

				s.depthSrv.Delete(si)
				srv.Stop()
			}
		} else {
			s.lastGetDepth.Store(si, time.Now())
		}

		return true
	})
}

func (s *Service) Klines(symbol, interval string) []*Kline {
	si := NewSymbolInterval(s.class, symbol, interval)
	srv, loaded := s.klinesSrv.Load(*si)
	if !loaded {
		if srv, loaded = s.klinesSrv.LoadOrStore(*si, NewKlinesSrv(s.ctx, si)); loaded == false {
			srv.(*KlinesSrv).Start()
		}
	}
	s.lastGetKlines.Store(*si, time.Now())

	return srv.(*KlinesSrv).GetKlines()
}

func (s *Service) Depth(symbol string) *Depth {
	si := NewSymbolInterval(s.class, symbol, "")
	srv, loaded := s.depthSrv.Load(*si)
	if !loaded {
		if srv, loaded = s.depthSrv.LoadOrStore(*si, NewDepthSrv(s.ctx, si)); loaded == false {
			srv.(*DepthSrv).Start()
		}
	}
	s.lastGetDepth.Store(*si, time.Now())

	return srv.(*DepthSrv).GetDepth()
}

// func (s *Service) autoRemoveDepthSrv(symbol string) *Depth {
// 	si := NewSymbolInterval(s.class, symbol, "")
// 	srv, loaded := s.klinesSrv.LoadOrStore(*si, NewDepthSrv(s.ctx, si))
// 	if loaded == false {
// 		srv.(*DepthSrv).Start()
// 	}

// 	return srv.(*DepthSrv).GetDepth()
// }
