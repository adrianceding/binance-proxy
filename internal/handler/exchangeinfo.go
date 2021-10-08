package handler

import (
	"net/http"
)

func (s *Handler) exchangeInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "apicache")
	w.Write(s.srv.ExchangeInfo())
}
