package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewFuturesHandler() func (w http.ResponseWriter, r *http.Request) {
	handler := &Futures{}
	return handler.Router
}


type Futures struct {
	Service string
}

func (t *Futures) Router(w http.ResponseWriter, r *http.Request) {
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

func (t *Futures) reverseProxy(w http.ResponseWriter, r *http.Request) {
	r.Host = "www.binancezh.io"
	u, _ := url.Parse("https://www.binancezh.io")
	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ServeHTTP(w, r)
}

func (t *Futures) klines(w http.ResponseWriter, r *http.Request) {

}

func (t *Futures) depth(w http.ResponseWriter, r *http.Request) {

}

func (t *Futures) price(w http.ResponseWriter, r *http.Request) {

}
