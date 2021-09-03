package handler

import (
	"binance-proxy/service/futures"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/davecgh/go-spew/spew"
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
	spew.Dump(s.srv.Klines("XMRUSDT", "5m"))
	// symbol	STRING	YES
	// interval	ENUM	YES	详见枚举定义：K线间隔
	// startTime	LONG	NO
	// endTime	LONG	NO
	// limit	INT	NO	默认 500; 最大 1000.
	// fmt.Println(s.klines("BTCUSDT", "5m"))
}

func (s *Futures) depth(w http.ResponseWriter, r *http.Request) {

}

func (s *Futures) price(w http.ResponseWriter, r *http.Request) {

}
