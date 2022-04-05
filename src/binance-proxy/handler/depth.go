package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *Handler) depth(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "20"
	}

	limitInt, err := strconv.Atoi(limit)
	switch {
	case err != nil, symbol == "", limitInt > 1000:
		s.reverseProxy(w, r)
		return
	}

	depth := s.srv.Depth(symbol)
	if depth == nil {
		s.reverseProxy(w, r)
		return
	}

	minLen := len(depth.Bids)
	if minLen > len(depth.Asks) {
		minLen = len(depth.Asks)
	}
	if minLen > limitInt {
		minLen = limitInt
	}

	bids := make([][2]string, minLen)
	asks := make([][2]string, minLen)
	for i := minLen; i > 0; i-- {
		asks[minLen-i] = [2]string{
			depth.Asks[minLen-i].Price,
			depth.Asks[minLen-i].Quantity,
		}
		bids[minLen-i] = [2]string{
			depth.Bids[minLen-i].Price,
			depth.Bids[minLen-i].Quantity,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(map[string]interface{}{
		"lastUpdateId": depth.LastUpdateID,
		"E":            depth.Time,
		"T":            depth.TradeTime,
		"bids":         bids,
		"asks":         asks,
	})
}
