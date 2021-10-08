package handler

import (
	"binance-proxy/internal/service"
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"

	log "github.com/sirupsen/logrus"
)

func NewHandler(ctx context.Context, class service.Class, enableFakeKline bool) func(w http.ResponseWriter, r *http.Request) {
	handler := &Handler{
		srv:             service.NewService(ctx, class),
		class:           class,
		enableFakeKline: enableFakeKline,
	}
	handler.ctx, handler.cancel = context.WithCancel(ctx)

	return handler.Router
}

type Handler struct {
	ctx    context.Context
	cancel context.CancelFunc

	class           service.Class
	srv             *service.Service
	enableFakeKline bool
}

func (s *Handler) Router(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v3/klines", "/fapi/v1/klines":
		s.klines(w, r)

	case "/api/v3/depth", "/fapi/v1/depth":
		s.depth(w, r)

	case "/api/v3/ticker/24hr":
		s.ticker(w, r)

	case "/api/v3/exchangeInfo", "/fapi/v1/exchangeInfo":
		s.exchangeInfo(w, r)

	default:
		s.reverseProxy(w, r)
	}
}

func (s *Handler) reverseProxy(w http.ResponseWriter, r *http.Request) {
	log.Debugf("%s reverse proxy.Path:%s", s.class, r.URL.RequestURI())

	service.RateWait(s.ctx, s.class, r.Method, r.URL.Path, r.URL.Query())

	var u *url.URL
	if s.class == service.SPOT {
		r.Host = "api.binance.com"
		u, _ = url.Parse("https://api.binance.com")
	} else {
		r.Host = "fapi.binance.com"
		u, _ = url.Parse("https://fapi.binance.com")
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ServeHTTP(w, r)
}
