package handler

import (
	"binance-proxy/service/spot"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

func NewSpotHandler() func(w http.ResponseWriter, r *http.Request) {
	handler := &Spot{
		srv: spot.NewSpot(),
	}
	return handler.Router
}

type Spot struct {
	srv *spot.Spot
}

func (s *Spot) Router(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v3/klines":
		s.klines(w, r)

	case "/api/v3/depth":
		s.depth(w, r)

	case "/api/v3/exchangeInfo":
		s.exchangeInfo(w, r)

	default:
		log.Printf("Spot reverse proxy.Path:%s", r.URL.Path)
		s.reverseProxy(w, r)
	}
}

func (s *Spot) reverseProxy(w http.ResponseWriter, r *http.Request) {
	r.Host = "api.binance.com"
	u, _ := url.Parse("https://api.binance.com")
	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ServeHTTP(w, r)
}

func (s *Spot) exchangeInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "apicache")
	w.Write(s.srv.ExchangeInfo())
}

func (s *Spot) klines(w http.ResponseWriter, r *http.Request) {
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

		// Do not forward. So as not to affect normal requests
		w.Write([]byte(`{"code": -1103,"msg": "Not support startTime and endTime.Symbol and interval is required.Limit must between 0 and 1500."}`))
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

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")
	j, _ := json.Marshal(klines)
	w.Write(j)
}

func (s *Spot) depth(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "20"
	}

	limitInt, err := strconv.Atoi(limit)
	switch {
	case err != nil, symbol == "", limitInt < 5, limitInt > 20:
		// Do not forward. So as not to affect normal requests
		w.Write([]byte(`{"code": -1120,"msg": "Symbol is required.Limit must between 5 and 20."}`))

		return
	}

	data := s.srv.Depth(symbol)
	minLen := len(data.Bids)
	if minLen > len(data.Asks) {
		minLen = len(data.Asks)
	}
	if minLen > limitInt {
		minLen = limitInt
	}

	bids := make([][2]string, minLen)
	asks := make([][2]string, minLen)
	for i := minLen; i > 0; i-- {
		asks[minLen-i] = [2]string{
			data.Asks[minLen-i].Price,
			data.Asks[minLen-i].Quantity,
		}
		bids[minLen-i] = [2]string{
			data.Bids[minLen-i].Price,
			data.Bids[minLen-i].Quantity,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")
	j, _ := json.Marshal(map[string]interface{}{
		"lastUpdateId": data.LastUpdateID,
		"E":            time.Now().UnixNano() / 1e6,
		"T":            time.Now().UnixNano() / 1e6,
		"bids":         bids,
		"asks":         asks,
	})
	w.Write(j)
}
