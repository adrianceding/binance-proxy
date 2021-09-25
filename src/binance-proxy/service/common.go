package service

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
