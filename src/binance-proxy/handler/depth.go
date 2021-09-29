package handler

import (
	"bytes"
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
	case err != nil, symbol == "", limitInt < 5, limitInt > 20:
		s.reverseProxy(w, r)
		return
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	buf.Reset()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Data-Source", "websocket")

	s.srv.Depth(symbol, limitInt, buf)

	w.Write(buf.Bytes())
}
