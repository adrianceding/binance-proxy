package spot

import (
	"log"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2"
)

type SpotDepth struct {
	mutex sync.RWMutex

	stopC chan struct{}
	si    SymbolInterval
	depth *client.DepthResponse
}

func NewFutresDepth(si SymbolInterval) *SpotDepth {
	s := &SpotDepth{si: si, stopC: make(chan struct{}, 1)}
	s.start()
	return s
}

func (s *SpotDepth) start() {
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
			doneC, stopC, err := client.WsPartialDepthServe100Ms(s.si.Symbol, "20", s.wsHandler, s.errHandler)
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

func (s *SpotDepth) Stop() {
	s.stopC <- struct{}{}
}

func (s *SpotDepth) wsHandler(event *client.WsPartialDepthEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	s.depth = &client.DepthResponse{
		LastUpdateID: event.LastUpdateID,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
}

func (s *SpotDepth) errHandler(err error) {
	log.Printf("%s.Depth websocket throw error!Error:%s", s.si, err)
}

func (s *SpotDepth) GetDepth() client.DepthResponse {
	defer s.mutex.RUnlock()
	s.mutex.RLock()

	var t client.DepthResponse
	if s.depth != nil {
		t = *s.depth
	}

	return t
}
