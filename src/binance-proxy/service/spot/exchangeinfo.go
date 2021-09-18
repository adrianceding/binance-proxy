package spot

import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"binance-proxy/tool"

	log "github.com/sirupsen/logrus"
)

type ExchangeInfoSrv struct {
	rw  sync.RWMutex
	ctx context.Context

	refreshDur   time.Duration
	url          string
	si           SymbolInterval
	exchangeInfo []byte
	updateTime   time.Time
}

func NewExchangeInfoSrv(ctx context.Context, si SymbolInterval) *ExchangeInfoSrv {
	return &ExchangeInfoSrv{
		ctx:        ctx,
		si:         si,
		url:        "https://api.binance.com/api/v3/exchangeInfo",
		refreshDur: 60 * time.Second,
	}
}

func (s *ExchangeInfoSrv) Start() {
	s.rw.Lock()
	defer s.rw.Unlock()

	if s.exchangeInfo != nil {
		return
	}

	for d := tool.NewDelayIterator(); ; d.Delay() {
		if s.refreshExchangeInfo() == nil {
			break
		}
	}

	go func() {
		rTimer := time.NewTimer(s.refreshDur)
		for {
			for d := tool.NewDelayIterator(); ; d.Delay() {
				if s.refreshExchangeInfo() == nil {
					break
				}
			}

			if !rTimer.Stop() {
				<-rTimer.C
			}
			rTimer.Reset(s.refreshDur)
			select {
			case <-s.ctx.Done():
				return
			case <-rTimer.C:
			}
		}
	}()
}

func (s *ExchangeInfoSrv) GetExchangeInfo() []byte {
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.exchangeInfo
}

func (s *ExchangeInfoSrv) refreshExchangeInfo() error {
	resp, err := http.Get(s.url)
	if err != nil {
		log.Errorf("Spot exchangeInfo init error!Error:%s", err)
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	s.exchangeInfo = data
	s.updateTime = time.Now()

	return nil
}
