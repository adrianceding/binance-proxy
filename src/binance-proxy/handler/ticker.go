package handler

import (
	"encoding/json"
	"net/http"
)

func (s *Handler) ticker(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")

	if symbol == "" {
		s.reverseProxy(w, r)
		return
	}
	ticker := s.srv.Ticker(symbol)
	if ticker == nil {
		s.reverseProxy(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(ticker)
}
