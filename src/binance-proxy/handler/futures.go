package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewFuturesHandler() func(w http.ResponseWriter, r *http.Request) {
	handler := &Futures{
		Service: service.NewFutures(),
	}
	return handler.Router
}

type Futures struct {
	Service service.Futures
}

func (s *Futures) Router(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/fapi/v1/klines":
		t.klines(w, r)
	case "/fapi/v1/ticker/depth":
		t.depth(w, r)
	case "/fapi/v1/ticker/24hr":
		t.price(w, r)
	default:
		t.reverseProxy(w, r)
	}
}

func (s *Futures) reverseProxy(w http.ResponseWriter, r *http.Request) {
	r.Host = "www.binancezh.io"
	u, _ := url.Parse("https://www.binancezh.io")
	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ServeHTTP(w, r)
}

func (s *Futures) klines(w http.ResponseWriter, r *http.Request) {
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
