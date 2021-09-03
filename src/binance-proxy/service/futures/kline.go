package futures

import (
	"fmt"

	client "github.com/adshao/go-binance/v2/futures"
)

type FuturesKlines struct {
	Klines [1001]FuturesKline
}
type FuturesKline struct {
	OpenDate         int64  // 1499040000000,      // 开盘时间
	OpenPrice        string // "0.01634790",       // 开盘价
	HighPrice        string // "0.80000000",       // 最高价
	LowPrice         string // "0.01575800",       // 最低价
	ClosePrice       string // "0.01577100",       // 收盘价(当前K线未结束的即为最新价)
	Volume           string // "148976.11427815",  // 成交量
	CloseDate        int64  // 1499644799999,      // 收盘时间
	QuoteVolume      string // "2434.19055334",    // 成交额
	TradeNum         int64  // 308,                // 成交笔数
	TakerVolume      string // "1756.87402397",    // 主动买入成交量
	TakerQuoteVolume string // "28.46694368",      // 主动买入成交额
	Ignore           string // "17928899.62484339" // 请忽略该参数
}

func NewFutresKlines(si SymbolInterval) (*FuturesKlines, error) {
	k := &FuturesKlines{}

	if _, _, err := client.WsAggTradeServe(si.Symbol, k.wsHandler, k.errHandler); err != nil {
		fmt.Println("init error:", err)
		return nil, err
	}

	return k, nil
}

func (s *FuturesKlines) wsHandler(event *client.WsAggTradeEvent) {
	event.
		fmt.Println("event:", event)
}

func (s *FuturesKlines) errHandler(err error) {
	fmt.Println("error:", err)
}
