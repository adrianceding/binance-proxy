package handler

import (
	"binance-proxy/service/futures"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
)

func NewFuturesHandler() func(w http.ResponseWriter, r *http.Request) {
	handler := &Futures{
		srv: futures.NewFutures(),
	}
	return handler.Router
}

type Futures struct {
	srv *futures.Futures
}

func (s *Futures) Router(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/fapi/v1/klines":
		s.klines(w, r)
	case "/fapi/v1/ticker/depth":
		s.depth(w, r)
	case "/fapi/v1/ticker/24hr":
		s.price(w, r)
	default:
		s.reverseProxy(w, r)
	}
}

func (s *Futures) reverseProxy(w http.ResponseWriter, r *http.Request) {
	r.Host = "www.binancezh.io"
	u, _ := url.Parse("https://www.binancezh.io")
	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ServeHTTP(w, r)
}

func (s *Futures) klines(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	interval := r.URL.Query().Get("interval")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "500"
	}

	limitInt, err := strconv.Atoi(limit)
	switch {
	case err != nil,
		limitInt <= 0,
		limitInt > 1500,
		r.URL.Query().Get("startTime") != "",
		r.URL.Query().Get("endTime") != "",
		symbol == "",
		interval == "":
		s.reverseProxy(w, r)
		return
	}

	data := s.srv.Klines(symbol, interval)
	if data == nil {
		s.reverseProxy(w, r)
		return
	}

}

func (s *Futures) depth(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "500"
	}

	limitInt, err := strconv.Atoi(limit)
	switch {
	case err != nil,
		symbol == "",
		limitInt <= 0,
		limitInt > 1000:
		s.reverseProxy(w, r)
		return
	}

	data := s.srv.Depth(symbol)
	if data == nil {
		s.reverseProxy(w, r)
		return
	}
}

func (s *Futures) price(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		s.reverseProxy(w, r)
	}

	data := s.srv.Price(symbol)
	if data == nil {
		s.reverseProxy(w, r)
		return
	}
}
