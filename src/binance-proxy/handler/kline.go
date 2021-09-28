package handler

import (
	"binance-proxy/service"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func (s *Handler) klines(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	interval := r.URL.Query().Get("interval")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "500"
	}
	limitInt, err := strconv.Atoi(limit)

	switch {
	case err != nil, limitInt <= 0, limitInt > 1000,
		r.URL.Query().Get("startTime") != "", r.URL.Query().Get("endTime") != "",
		symbol == "", interval == "":

		s.reverseProxy(w, r)
		return
	}

	rawKlines := s.srv.Klines(symbol, interval)
	minLen := len(rawKlines)
	if minLen > limitInt {
		minLen = limitInt
	}

	klines := rawKlines[len(rawKlines)-minLen:]
	if len(rawKlines) > 0 && time.Now().UnixNano()/1e6 > rawKlines[len(rawKlines)-1][service.K_CloseTime].(int64) {
		lastK := rawKlines[len(rawKlines)-1]
		closeTime := lastK[service.K_CloseTime].(int64)
		openTime := lastK[service.K_OpenTime].(int64)
		close := lastK[service.K_Close].(string)

		klines = append(klines, &service.Kline{
			service.K_OpenTime:                 closeTime + 1,
			service.K_Open:                     close,
			service.K_High:                     close,
			service.K_Low:                      close,
			service.K_Close:                    close,
			service.K_Volume:                   "0.0",
			service.K_CloseTime:                closeTime + 1 + (closeTime - openTime),
			service.K_QuoteAssetVolume:         "0.0",
			service.K_TradeNum:                 0,
			service.K_TakerBuyBaseAssetVolume:  "0.0",
			service.K_TakerBuyQuoteAssetVolume: "0.0",
			service.K_NoUse:                    "0",
		})
		klines = klines[len(klines)-minLen:]
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(klines)
}
