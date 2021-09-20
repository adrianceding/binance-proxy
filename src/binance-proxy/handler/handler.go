package handler

import (
	"binance-proxy/service"
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"

	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

func NewHandler(ctx context.Context, class service.Class) func(w http.ResponseWriter, r *http.Request) {
	handler := &Handler{
		srv:   service.NewService(ctx, class),
		class: class,
	}
	if class == service.SPOT {
		handler.limiter = rate.NewLimiter(20, 1200)
	} else {
		handler.limiter = rate.NewLimiter(40, 2400)
	}

	return handler.Router
}

type Handler struct {
	class   service.Class
	srv     *service.Service
	limiter *rate.Limiter
}

func (s *Handler) Router(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v3/klines", "/fapi/v1/klines":
		s.klines(w, r)

	case "/api/v3/depth", "/fapi/v1/depth":
		s.depth(w, r)

	case "/api/v3/exchangeInfo", "/fapi/v1/exchangeInfo":
		s.exchangeInfo(w, r)

	default:
		log.Debugf("%s reverse proxy.Path:%s", s.class, r.URL.Path)
		s.reverseProxy(w, r)
	}
}

func (s *Handler) reverseProxy(w http.ResponseWriter, r *http.Request) {
	var u *url.URL
	if s.class == service.SPOT {
		r.Host = "api.binance.com"
		u, _ = url.Parse("https://api.binance.com")
	} else {
		r.Host = "fapi.binance.com"
		u, _ = url.Parse("https://fapi.binance.com")
	}

	if s.class == service.SPOT {
		s.limiter.WaitN(context.Background(), 1)
	} else {
		s.limiter.WaitN(context.Background(), 5)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ServeHTTP(w, r)
}
