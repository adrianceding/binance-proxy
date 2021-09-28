package handler

import (
	"bytes"
	"net/http"
	"strconv"
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

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer pool.Put(buf)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")

	s.srv.Klines(symbol, interval, true, limitInt, buf)
	w.Write(buf.Bytes())
}
