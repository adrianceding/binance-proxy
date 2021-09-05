package futures

import (
	"log"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
)

type FuturesDepth struct {
	mutex sync.RWMutex

	SI    SymbolInterval
	Depth *client.DepthResponse
}

func NewFutresDepth(si SymbolInterval) (*FuturesDepth, error) {
	k := &FuturesDepth{
		SI: si,
	}
	if err := k.initWs(); err != nil {
		return nil, err
	}

	return k, nil
}

func (s *FuturesDepth) initWs() error {
	client.WebsocketKeepalive = true
	if _, _, err := client.WsPartialDepthServeWithRate(s.SI.Symbol, 20, 100*time.Millisecond, s.wsHandler, s.errHandler); err != nil {
		log.Println("depth init ws ", s.SI, " error:", err)
		return err
	}
	log.Println("depth init ws ", s.SI, " success.")

	return nil
}

func (s *FuturesDepth) wsHandler(event *client.WsDepthEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	s.Depth = &client.DepthResponse{
		LastUpdateID: event.LastUpdateID,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
}

func (s *FuturesDepth) errHandler(err error) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	log.Println("depth ws err handler ", s.SI, ".error:", err)

	s.Depth = nil

	for {
		if s.initWs() == nil {
			break
		}
		time.Sleep(time.Second)
	}
}
