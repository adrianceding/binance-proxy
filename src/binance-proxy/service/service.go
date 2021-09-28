package service

import (
	"bytes"
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

func (s *Service) Klines(symbol, interval string, fakeKline bool, limitInt int, buf *bytes.Buffer) {
	si := NewSymbolInterval(s.class, symbol, interval)
	srv, loaded := s.klinesSrv.LoadOrStore(*si, NewKlinesSrv(s.ctx, si))
	if loaded == false {
		srv.(*KlinesSrv).Start()
	}

	// 	if len(rawKlines) > 0 && time.Now().UnixNano()/1e6 > rawKlines[len(rawKlines)-1][service.K_CloseTime].(int64) {
	// 	lastK := rawKlines[len(rawKlines)-1]
	// 	closeTime := lastK[service.K_CloseTime].(int64)
	// 	openTime := lastK[service.K_OpenTime].(int64)
	// 	close := lastK[service.K_Close].(string)

	// 	klines = append(klines, &service.Kline{
	// 		service.K_OpenTime:                 closeTime + 1,
	// 		service.K_Open:                     close,
	// 		service.K_High:                     close,
	// 		service.K_Low:                      close,
	// 		service.K_Close:                    close,
	// 		service.K_Volume:                   "0.0",
	// 		service.K_CloseTime:                closeTime + 1 + (closeTime - openTime),
	// 		service.K_QuoteAssetVolume:         "0.0",
	// 		service.K_TradeNum:                 0,
	// 		service.K_TakerBuyBaseAssetVolume:  "0.0",
	// 		service.K_TakerBuyQuoteAssetVolume: "0.0",
	// 		service.K_NoUse:                    "0",
	// 	})
	// 	klines = klines[len(klines)-minLen:]
	// }

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
