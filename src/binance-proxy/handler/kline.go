package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"binance-proxy/service"
)

func (s *Handler) klines(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	interval := r.URL.Query().Get("interval")
	limitInt, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limitInt == 0 {
		limitInt = 500
	}

	startTimeUnix, _ := strconv.Atoi(r.URL.Query().Get("startTime"))
	startTime := time.Unix(int64(startTimeUnix/1000), 0)

	switch {
	case limitInt <= 0, limitInt > 1000,
		startTime.Unix() > 0 && startTime.Before(time.Now().Add(service.INTERVAL_2_DURATION[interval]*999*-1)),
		r.URL.Query().Get("endTime") != "",
		symbol == "", interval == "":
		s.reverseProxy(w, r)
		return
	}

	data := s.srv.Klines(symbol, interval)
	klines := make([]interface{}, 0)
	startTimeUnixMs := startTime.Unix() * 1000
	if startTimeUnixMs == 0 && limitInt < len(data) {
		data = data[len(data)-limitInt:]
	}
	for _, v := range data {
		if len(klines) >= limitInt {
			break
		}

		if startTimeUnixMs > 0 && startTimeUnixMs > v.OpenTime {
			continue
		}

		klines = append(klines, []interface{}{
			v.OpenTime,
			v.Open,
			v.High,
			v.Low,
			v.Close,
			v.Volume,
			v.CloseTime,
			v.QuoteAssetVolume,
			v.TradeNum,
			v.TakerBuyBaseAssetVolume,
			v.TakerBuyQuoteAssetVolume,
			"0",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")
	j, _ := json.Marshal(klines)
	w.Write(j)
}
