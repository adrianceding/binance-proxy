package spot

import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type SpotExchangeInfoSrv struct {
	rw  sync.RWMutex
	ctx context.Context

	si           SymbolInterval
	exchangeInfo []byte
	updateTime   time.Time
}

func NewSpotExchangeInfoSrv(ctx context.Context, si SymbolInterval) *SpotExchangeInfoSrv {
	return &SpotExchangeInfoSrv{ctx: ctx, si: si}
}

func (s *SpotExchangeInfoSrv) Start() {
	defer s.rw.Unlock()
	s.rw.Lock()

	s.onceStart.Do(func() {
		go func() {
			rDur := 5 * time.Second
			rTimer := time.NewTimer(rDur)
			for {
				for delay := 1; ; delay *= 2 {
					if delay > 60 {
						delay = 60
					}
					time.Sleep(time.Duration(delay-1) * time.Second)

					data, err := s.getExchangeInfo()
					if err != nil {
						log.Errorf("Spot exchangeInfo init error!Error:%s", err)
						continue
					}

					s.rw.Lock()
					s.exchangeInfo = data
					s.updateTime = time.Now()
					s.rw.Unlock()

					s.onceInited.Do(func() {
						log.Debugf("Spot exchangeInfo update success!")
						close(s.inited)
					})

					break
				}

				if !rTimer.Stop() {
					<-rTimer.C
				}
				rTimer.Reset(rDur)
				select {
				case <-s.stopC:
					return
				case <-rTimer.C:
				}
			}
		}()
	})
}

func (s *SpotExchangeInfoSrv) GetExchangeInfo() []byte {
	defer s.rw.RUnlock()
	s.rw.RLock()

	return s.exchangeInfo
}

func (s *SpotExchangeInfoSrv) refreshExchangeInfo() error {

}

func (s *SpotExchangeInfoSrv) getExchangeInfo() ([]byte, error) {
	resp, err := http.Get("https://api.binance.com/api/v3/exchangeInfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
