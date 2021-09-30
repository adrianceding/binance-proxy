package service

type symbolInterval struct {
	Class    Class
	Symbol   string
	Interval string
}
type Class string

var FUTURES Class = "FUTURES"
var SPOT Class = "SPOT"

func NewSymbolInterval(class Class, symbol, interval string) symbolInterval {
	return symbolInterval{Class: class, Symbol: symbol, Interval: interval}
}
