package spot

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	client "github.com/adshao/go-binance/v2"
)

type SpotDepthSrv struct {
	mutex sync.RWMutex

	onceStart  sync.Once
	onceInited sync.Once
	onceStop   sync.Once

	inited chan struct{}
	stopC  chan struct{}
	errorC chan struct{}

	si         SymbolInterval
	depth      *client.DepthResponse
	updateTime time.Time
}

func NewSpotDepthSrv(si SymbolInterval) *SpotDepthSrv {
	return &SpotDepthSrv{
		si:     si,
		stopC:  make(chan struct{}, 1),
		inited: make(chan struct{}),
		errorC: make(chan struct{}, 1),
	}
}

func (s *SpotDepthSrv) Start() {
	go func() {
		for delay := 1; ; delay *= 2 {
			s.mutex.Lock()
			s.depth = nil
			s.mutex.Unlock()

			if delay > 60 {
				delay = 60
			}
			time.Sleep(time.Second * time.Duration(delay-1))

			doneC, stopC, err := client.WsPartialDepthServe100Ms(s.si.Symbol, "20", s.wsHandler, s.errHandler)
			if err != nil {
				log.Errorf("%s.Spot websocket depth connect error!Error:%s", s.si, err)
				continue
			}

			delay = 1
			select {
			case <-s.stopC:
				stopC <- struct{}{}
				return
			case <-doneC:
			case <-s.errorC:
			}
			log.Debugf("%s.Spot websocket depth disconnected!Reconnecting", s.si)
		}
	}()
}

func (s *SpotDepthSrv) Stop() {
	s.onceStop.Do(func() {
		s.stopC <- struct{}{}
	})
}

func (s *SpotDepthSrv) wsHandler(event *client.WsPartialDepthEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	s.depth = &client.DepthResponse{
		LastUpdateID: event.LastUpdateID,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
	s.updateTime = time.Now()

	s.onceInited.Do(func() {
		log.Debugf("%s.Spot depth init success!", s.si)
		close(s.inited)
	})
}

func (s *SpotDepthSrv) errHandler(err error) {
	log.Errorf("%s.Spot depth websocket throw error!Error:%s", s.si, err)
	s.errorC <- struct{}{}
}

func (s *SpotDepthSrv) GetDepth() (*client.DepthResponse, time.Time) {
	<-s.inited

	defer s.mutex.Unlock()
	s.mutex.Lock()

	return s.depth, s.updateTime
}
