package futures

import (
	"log"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
)

type FuturesDepth struct {
	mutex sync.RWMutex

	stopC chan struct{}
	si    SymbolInterval
	depth *client.DepthResponse
}

func NewFutresDepth(si SymbolInterval) *FuturesDepth {
	s := &FuturesDepth{si: si, stopC: make(chan struct{}, 1)}
	s.start()
	return s
}

func (s *FuturesDepth) start() {
	go func() {
		for delay := 1; ; delay *= 2 {
			if delay > 60 {
				delay = 60
			}
			time.Sleep(time.Second * time.Duration(delay-1))

			s.mutex.Lock()
			s.depth = nil
			s.mutex.Unlock()

			client.WebsocketKeepalive = true
			doneC, stopC, err := client.WsPartialDepthServeWithRate(s.si.Symbol, 20, 100*time.Millisecond, s.wsHandler, s.errHandler)
			if err != nil {

				continue
			}

			select {
			case stopC <- <-s.stopC:
				return
			case <-doneC:
			}
		}
	}()
}

func (s *FuturesDepth) Stop() {
	s.stopC <- struct{}{}
}

func (s *FuturesDepth) wsHandler(event *client.WsDepthEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	s.depth = &client.DepthResponse{
		LastUpdateID: event.LastUpdateID,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
}

func (s *FuturesDepth) errHandler(err error) {
	log.Printf("%s.Depth websocket throw error!Error:%s", s.si, err)
}

func (s *FuturesDepth) GetDepth() client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	var t client.DepthResponse
	if s.depth != nil {
		t = *s.depth
	}

	return t
}
