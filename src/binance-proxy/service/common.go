package service

import "time"

var INTERVAL_2_DURATION = map[string]time.Duration{
	"1m":  1 * time.Minute,
	"3m":  3 * time.Minute,
	"5m":  5 * time.Minute,
	"15m": 15 * time.Minute,
	"30m": 30 * time.Minute,
	"1h":  1 * time.Hour,
	"2h":  2 * time.Hour,
	"4h":  4 * time.Hour,
	"6h":  6 * time.Hour,
	"8h":  8 * time.Hour,
	"12h": 12 * time.Hour,
	"1d":  1 * 24 * time.Hour,
	"3d":  3 * 24 * time.Hour,
	"1w":  7 * 24 * time.Hour,
	"1M":  31 * 24 * time.Hour,
}

type symbolInterval struct {
	Class    Class
	Symbol   string
	Interval string
}
type Class int8

var SPOT Class = 0
var FUTURES Class = 1

func NewSymbolInterval(class Class, symbol, interval string) *symbolInterval {
	return &symbolInterval{Class: class, Symbol: symbol, Interval: interval}
}
