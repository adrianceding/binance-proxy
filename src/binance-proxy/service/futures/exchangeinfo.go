package futures

import (
	"context"
	"time"

	"binance-proxy/spot"
)

type ExchangeInfoSrv = spot.ExchangeInfoSrv

func NewExchangeInfoSrv(ctx context.Context, si SymbolInterval) *ExchangeInfoSrv {
	return &ExchangeInfoSrv{
		ctx:        ctx,
		si:         si,
		url:        "https://fapi.binance.com/fapi/v1/exchangeInfo",
		refreshDur: 60 * time.Second,
	}
}
