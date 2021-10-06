package service

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/time/rate"
)

var (
	SpotLimiter    = rate.NewLimiter(20, 1200)
	FuturesLimiter = rate.NewLimiter(40, 2400)
)

func RateWait(ctx context.Context, class Class, method, path string, query url.Values) {
	weight := 1
	switch path {
	case "/fapi/v1/klines":
		weight = 5
		limitInt, _ := strconv.Atoi(query.Get("limit"))
		if limitInt >= 1 && limitInt < 100 {
			weight = 1
		} else if limitInt >= 100 && limitInt < 500 {
			weight = 2
		} else if limitInt >= 500 && limitInt <= 1000 {
			weight = 5
		} else if limitInt > 1000 && limitInt <= 1500 {
			weight = 10
		}
	case "/api/v3/depth":
		limitInt, _ := strconv.Atoi(query.Get("limit"))
		if limitInt >= 5 && limitInt <= 100 {
			weight = 1
		} else if limitInt >= 100 && limitInt < 500 {
			weight = 2
		} else if limitInt == 500 {
			weight = 5
		} else if limitInt == 1000 {
			weight = 10
		} else if limitInt == 5000 {
			weight = 50
		}
	case "/fapi/v1/depth":
		limitInt, _ := strconv.Atoi(query.Get("limit"))
		if limitInt >= 5 && limitInt <= 50 {
			weight = 2
		} else if limitInt == 100 {
			weight = 5
		} else if limitInt == 500 {
			weight = 10
		} else if limitInt == 1000 {
			weight = 20
		}
	case "/api/v3/ticker/24hr", "/fapi/v1/ticker/24hr":
		if query.Get("symbol") == "" {
			weight = 40
		}
	case "/api/v3/exchangeInfo", "/fapi/v1/exchangeInfo", "/api/v3/account", "/api/v3/myTrades":
		weight = 10
	case "/api/v3/order":
		if method == http.MethodGet {
			weight = 2
		}
	case "/fapi/v1/userTrades", "/fapi/v2/account":
		weight = 5

	}

	if class == SPOT {
		SpotLimiter.WaitN(ctx, weight)
	} else {
		FuturesLimiter.WaitN(ctx, weight)
	}
}
