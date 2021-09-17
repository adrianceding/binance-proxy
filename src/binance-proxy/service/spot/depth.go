package spot

import (
	"log"
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2"
)

type SpotDepth struct {
	mutex sync.RWMutex

	onceStart  sync.Once
	onceInited sync.Once
	onceStop   sync.Once

	inited chan struct{}
	stopC  chan struct{}

	si         SymbolInterval
	depth      *client.DepthResponse
	updateTime time.Time
}

func NewSpotDepth(si SymbolInterval) *SpotDepth {
	return &SpotDepth{
		si:     si,
		stopC:  make(chan struct{}, 1),
		inited: make(chan struct{}),
	}
}

func (s *SpotDepth) Start() {
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

			delay = 1
			select {
			case stopC <- <-s.stopC:
				return
			case <-doneC:
			}
		}
	}()
}

func (s *SpotDepth) Stop() {
	s.onceStop.Do(func() {
		s.stopC <- struct{}{}
	})
}

func (s *SpotDepth) wsHandler(event *client.WsPartialDepthEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	s.depth = &client.DepthResponse{
		LastUpdateID: event.LastUpdateID,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
	s.updateTime = time.Now()

	s.onceInited.Do(func() {
		close(s.inited)
	})
}

func (s *SpotDepth) errHandler(err error) {
	log.Printf("%s.Depth websocket throw error!Error:%s", s.si, err)
}

func (s *SpotDepth) GetDepth() client.DepthResponse {
	<-s.inited

	defer s.mutex.RUnlock()
	s.mutex.RLock()

	var t client.DepthResponse
	// if time.Now().Sub(s.updateTime).Seconds() > 5 {
	// 	return t
	// }
	if s.depth != nil {
		t = *s.depth
	}

	return t
}
