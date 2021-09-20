package service

import "golang.org/x/time/rate"

var SpotLimiter = rate.NewLimiter(20, 1200)
var FuturesLimiter = rate.NewLimiter(40, 2400)
