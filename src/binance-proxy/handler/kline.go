package handler

import (
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

	data := s.srv.Klines(symbol, interval)
	minLen := len(data)
	if minLen > limitInt {
		minLen = limitInt
	}

	klines := make([]interface{}, minLen)
	for i := minLen; i > 0; i-- {
		ri := len(data) - i
		klines[minLen-i] = []interface{}{
			data[ri].OpenTime,
			data[ri].Open,
			data[ri].High,
			data[ri].Low,
			data[ri].Close,
			data[ri].Volume,
			data[ri].CloseTime,
			data[ri].QuoteAssetVolume,
			data[ri].TradeNum,
			data[ri].TakerBuyBaseAssetVolume,
			data[ri].TakerBuyQuoteAssetVolume,
			"0",
		}
	}

	if len(data) > 0 && time.Now().UnixNano()/1e6 > data[len(data)-1].CloseTime {
		klines = append(klines, []interface{}{
			data[len(data)-1].CloseTime + 1,
			data[len(data)-1].Close,
			data[len(data)-1].Close,
			data[len(data)-1].Close,
			data[len(data)-1].Close,
			"0.0",
			data[len(data)-1].CloseTime + 1 + (data[len(data)-1].CloseTime - data[len(data)-1].OpenTime),
			"0.0",
			0,
			"0.0",
			"0.0",
			"0",
		})
		klines = klines[len(klines)-minLen:]
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")
	j, _ := json.Marshal(klines)
	w.Write(j)
}
