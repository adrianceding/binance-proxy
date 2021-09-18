package futures

import (
	"sync"
	"time"

	client "github.com/adshao/go-binance/v2/futures"
	log "github.com/sirupsen/logrus"
)

type FuturesDepthSrv struct {
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

func NewFuturesDepthSrv(si SymbolInterval) *FuturesDepthSrv {
	return &FuturesDepthSrv{
		si:     si,
		stopC:  make(chan struct{}, 1),
		inited: make(chan struct{}),
		errorC: make(chan struct{}, 1),
	}
}

func (s *FuturesDepthSrv) GetDepth() (*client.DepthResponse, time.Time) {
	<-s.inited

	return s.depth, s.updateTime
}

func (s *FuturesDepthSrv) Stop() {
	s.onceStop.Do(func() {
		s.stopC <- struct{}{}
	})
}

func (s *FuturesDepthSrv) Start() {
	s.onceStart.Do(func() {
		go func() {
			for delay := 1; ; delay *= 2 {
				s.mutex.Lock()
				s.depth = nil
				s.mutex.Unlock()

				if delay > 60 {
					delay = 60
				}
				time.Sleep(time.Second * time.Duration(delay-1))

				doneC, stopC, err := client.WsPartialDepthServeWithRate(s.si.Symbol, 20, 100*time.Millisecond, s.wsHandler, s.errHandler)
				if err != nil {
					log.Errorf("%s.Futures websocket depth connect error!Error:%s", s.si, err)
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

				log.Debugf("%s.Futures websocket depth disconnected!Reconnecting", s.si)
			}
		}()
	})
}

func (s *FuturesDepthSrv) wsHandler(event *client.WsDepthEvent) {
	defer s.mutex.Unlock()
	s.mutex.Lock()

	s.depth = &client.DepthResponse{
		LastUpdateID: event.LastUpdateID,
		Bids:         event.Bids,
		Asks:         event.Asks,
	}
	s.updateTime = time.Now()

	s.onceInited.Do(func() {
		log.Debugf("%s.Futures depth init success!", s.si)
		close(s.inited)
	})
}

func (s *FuturesDepthSrv) errHandler(err error) {
	log.Errorf("%s.Futures depth websocket throw error!Error:%s", s.si, err)
	s.errorC <- struct{}{}
}
