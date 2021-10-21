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

	startTickerWithKline bool
	startDepthWithKline  bool

	class           Class
	exchangeInfoSrv *ExchangeInfoSrv
	klinesSrv       sync.Map // map[symbolInterval]*Klines
	depthSrv        sync.Map // map[symbolInterval]*Depth
	tickerSrv       sync.Map // map[symbolInterval]*Ticker

	lastGetKlines sync.Map // map[symbolInterval]time.Time
	lastGetDepth  sync.Map // map[symbolInterval]time.Time
	lastGetTicker sync.Map // map[symbolInterval]time.Time
}

func NewService(ctx context.Context, class Class, startTickerWithKline, startDepthWithKline bool) *Service {
	s := &Service{
		class:                class,
		startTickerWithKline: startTickerWithKline,
		startDepthWithKline:  startDepthWithKline,
	}
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

func (s *Service) autoRemoveExpired() {
	var aliveKlines = make(map[string]struct{})
	s.klinesSrv.Range(func(k, v interface{}) bool {
		si := k.(symbolInterval)
		srv := v.(*KlinesSrv)

		aliveKlines[si.Symbol] = struct{}{}

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
			_, isKlineAlive := aliveKlines[si.Symbol]

			if ((s.startDepthWithKline && !isKlineAlive) || !s.startDepthWithKline) && time.Now().Sub(t.(time.Time)) > 2*time.Minute {
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
	s.tickerSrv.Range(func(k, v interface{}) bool {
		si := k.(symbolInterval)
		srv := v.(*TickerSrv)

		if t, ok := s.lastGetTicker.Load(si); ok {
			_, isKlineAlive := aliveKlines[si.Symbol]

			if ((s.startTickerWithKline && !isKlineAlive) || !s.startTickerWithKline) && time.Now().Sub(t.(time.Time)) > 2*time.Minute {
				log.Debugf("%s.Ticker srv expired!Removed", si)
				s.lastGetTicker.Delete(si)

				s.tickerSrv.Delete(si)
				srv.Stop()
			}
		} else {
			s.lastGetTicker.Store(si, time.Now())
		}

		return true
	})
}

func (s *Service) Ticker(symbol string) *Ticker24hr {
	si := NewSymbolInterval(s.class, symbol, "")
	srv := s.StartTickerSrv(si)

	s.lastGetTicker.Store(*si, time.Now())

	return srv.GetTicker()
}

func (s *Service) ExchangeInfo() []byte {
	return s.exchangeInfoSrv.GetExchangeInfo()
}

func (s *Service) Klines(symbol, interval string) []*Kline {
	si := NewSymbolInterval(s.class, symbol, interval)
	srv := s.StartKlineSrv(si)
	if s.startTickerWithKline {
		s.StartTickerSrv(NewSymbolInterval(s.class, symbol, ""))
	}
	if s.startDepthWithKline {
		s.StartDepthSrv(NewSymbolInterval(s.class, symbol, ""))
	}

	s.lastGetKlines.Store(*si, time.Now())

	return srv.GetKlines()
}

func (s *Service) Depth(symbol string) *Depth {
	si := NewSymbolInterval(s.class, symbol, "")
	srv := s.StartDepthSrv(si)

	s.lastGetDepth.Store(*si, time.Now())

	return srv.GetDepth()
}

func (s *Service) StartKlineSrv(si *symbolInterval) *KlinesSrv {
	srv, loaded := s.klinesSrv.Load(*si)
	if !loaded {
		if srv, loaded = s.klinesSrv.LoadOrStore(*si, NewKlinesSrv(s.ctx, si)); loaded == false {
			srv.(*KlinesSrv).Start()
		}
	}
	return srv.(*KlinesSrv)
}

func (s *Service) StartDepthSrv(si *symbolInterval) *DepthSrv {
	srv, loaded := s.depthSrv.Load(*si)
	if !loaded {
		if srv, loaded = s.depthSrv.LoadOrStore(*si, NewDepthSrv(s.ctx, si)); loaded == false {
			srv.(*DepthSrv).Start()
		}
	}
	return srv.(*DepthSrv)
}

func (s *Service) StartTickerSrv(si *symbolInterval) *TickerSrv {
	srv, loaded := s.tickerSrv.Load(*si)
	if !loaded {
		if srv, loaded = s.tickerSrv.LoadOrStore(*si, NewTickerSrv(s.ctx, si)); loaded == false {
			srv.(*TickerSrv).Start()
		}
	}
	return srv.(*TickerSrv)
}
