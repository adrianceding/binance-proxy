package service

import "golang.org/x/time/rate"

var SpotLimiter = rate.NewLimiter(20, 600)
var FuturesLimiter = rate.NewLimiter(40, 1200)
