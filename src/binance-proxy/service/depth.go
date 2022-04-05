package service

import (
	"binance-proxy/tool"
	"container/list"
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	spot "github.com/adshao/go-binance/v2"
	futures "github.com/adshao/go-binance/v2/futures"
)

type PriceLevel struct {
	Price         string
	Quantity      string
	PriceFloat    float64
	QuantityFloat float64
}

type Depth struct {
	LastUpdateID int64
	Time         int64
	TradeTime    int64
	Bids         []PriceLevel
	Asks         []PriceLevel
}

type Side string

const (
	ASK Side = "ask"
	BID Side = "bid"
)

type DepthSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	si *symbolInterval

	LastUpdateID     int64
	Time             int64
	depthBidList     *list.List
	depthAskList     *list.List
	bidPrice2Element map[string]*list.Element
	askPrice2Element map[string]*list.Element
	depth            *Depth
}

func NewDepthSrv(ctx context.Context, si *symbolInterval) *DepthSrv {
	s := &DepthSrv{si: si}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.initCtx, s.initDone = context.WithCancel(context.Background())
	s.Reset()

	return s
}

func (s *DepthSrv) Reset() {
	s.rw.Lock()
	defer s.rw.Unlock()

	s.LastUpdateID = 0
	s.Time = 0
	s.depthBidList = list.New()
	s.depthAskList = list.New()
	s.bidPrice2Element = make(map[string]*list.Element)
	s.askPrice2Element = make(map[string]*list.Element)

	s.depth = nil
}

func (s *DepthSrv) Start() {
	go func() {
		for d := tool.NewDelayIterator(); ; d.Delay() {
			s.Reset()

			doneC, stopC, err := s.connect()
			if err != nil {
				log.Errorf("%s.Websocket depth connect error!Error:%s", s.si, err)
				continue
			}

			log.Debugf("%s.Websocket depth connect success!", s.si)
			select {
			case <-s.ctx.Done():
				stopC <- struct{}{}
				return
			case <-doneC:
			}

			log.Debugf("%s.Websocket depth disconnected!Reconnecting", s.si)
		}
	}()
}

func (s *DepthSrv) Stop() {
	s.cancel()
}

func (s *DepthSrv) errHandler(err error) {
	log.Errorf("%s.Depth websocket throw error!Error:%s", s.si, err)
}

func (s *DepthSrv) connect() (doneC, stopC chan struct{}, err error) {
	if s.si.Class == SPOT {
		return spot.WsDepthServe(s.si.Symbol,
			func(event *spot.WsDepthEvent) { s.wsHandler(event) },
			s.errHandler,
		)
	} else {
		return futures.WsDiffDepthServeWithRate(s.si.Symbol, 100*time.Millisecond,
			func(event *futures.WsDepthEvent) { s.wsHandler(event) },
			s.errHandler,
		)
	}
}

func (s *DepthSrv) initDepthData() {
	var depth interface{}
	var err error
	for d := tool.NewDelayIterator(); ; d.Delay() {
		if s.si.Class == SPOT {
			RateWait(s.ctx, s.si.Class, http.MethodGet, "/api/v3/depth", url.Values{
				"limit": []string{"1000"},
			})
			depth, err = spot.NewClient("", "").NewDepthService().
				Symbol(s.si.Symbol).Limit(1000).
				Do(s.ctx)
		} else {
			RateWait(s.ctx, s.si.Class, http.MethodGet, "/fapi/v1/depth", url.Values{
				"limit": []string{"1000"},
			})
			depth, err = futures.NewClient("", "").NewDepthService().
				Symbol(s.si.Symbol).Limit(1000).
				Do(s.ctx)
		}
		if err != nil {
			log.Errorf("%s.Get init depth error!Error:%s", s.si, err)
			continue
		}

		if v, ok := depth.(*spot.DepthResponse); ok {
			s.LastUpdateID = v.LastUpdateID
			s.Time = int64(time.Now().UnixNano() / 1e6)
			for _, v1 := range v.Asks {
				price, quantity, _ := v1.Parse()
				s.update(ASK, s.askPrice2Element, s.depthAskList, &PriceLevel{
					Price:         v1.Price,
					Quantity:      v1.Quantity,
					PriceFloat:    price,
					QuantityFloat: quantity,
				})
			}
			for _, v1 := range v.Bids {
				price, quantity, _ := v1.Parse()
				s.update(BID, s.bidPrice2Element, s.depthBidList, &PriceLevel{
					Price:         v1.Price,
					Quantity:      v1.Quantity,
					PriceFloat:    price,
					QuantityFloat: quantity,
				})
			}

		} else if v, ok := depth.(*futures.DepthResponse); ok {
			s.LastUpdateID = v.LastUpdateID
			s.Time = int64(time.Now().UnixNano() / 1e6)
			log.Debug(int64(time.Now().UnixNano()/1e6), time.Now().UnixNano())
			for _, v1 := range v.Asks {
				price, quantity, _ := v1.Parse()
				s.update(ASK, s.askPrice2Element, s.depthAskList, &PriceLevel{
					Price:         v1.Price,
					Quantity:      v1.Quantity,
					PriceFloat:    price,
					QuantityFloat: quantity,
				})
			}
			for _, v1 := range v.Bids {
				price, quantity, _ := v1.Parse()
				s.update(BID, s.bidPrice2Element, s.depthBidList, &PriceLevel{
					Price:         v1.Price,
					Quantity:      v1.Quantity,
					PriceFloat:    price,
					QuantityFloat: quantity,
				})
			}
		}

		s.convertDepth()

		defer s.initDone()

		break
	}
}

func (s *DepthSrv) update(side Side, m map[string]*list.Element, l *list.List, pl *PriceLevel) {
	if pl.QuantityFloat > 0 {
		if _, ok := m[pl.Price]; ok {
			m[pl.Price].Value = pl
		} else {
			var prevEle *list.Element
			for ele := l.Front(); ele != nil; ele = ele.Next() {
				if (side == ASK && ele.Value.(*PriceLevel).PriceFloat > pl.PriceFloat) || (side == BID && ele.Value.(*PriceLevel).PriceFloat < pl.PriceFloat) {
					prevEle = ele
					break
				}
			}
			if prevEle != nil {
				m[pl.Price] = l.InsertBefore(pl, prevEle)
			} else {
				m[pl.Price] = l.PushBack(pl)
			}
		}
	} else {
		if _, ok := m[pl.Price]; ok {
			l.Remove(m[pl.Price])
			delete(m, pl.Price)
		}
	}
}

func (s *DepthSrv) wsHandler(event interface{}) {
	if s.LastUpdateID == 0 {
		s.initDepthData()
	}

	if v, ok := event.(*spot.WsDepthEvent); ok {
		if v.LastUpdateID <= s.LastUpdateID {
			return
		} else if v.FirstUpdateID != s.LastUpdateID+1 && v.FirstUpdateID > s.LastUpdateID+1 {
			log.Errorf(
				"%s.last update id error!FirstUpdateID:%s,LastUpdateID:%s,LocalUpdateID:%s",
				s.si, v.FirstUpdateID, v.LastUpdateID, s.LastUpdateID,
			)
			s.Reset()
			return
		}

		s.LastUpdateID = v.LastUpdateID
		s.Time = int64(time.Now().UnixNano() / 1e6)
		for _, v1 := range v.Asks {
			price, quantity, _ := v1.Parse()
			s.update(ASK, s.askPrice2Element, s.depthAskList, &PriceLevel{
				Price:         v1.Price,
				Quantity:      v1.Quantity,
				PriceFloat:    price,
				QuantityFloat: quantity,
			})
		}
		for _, v1 := range v.Bids {
			price, quantity, _ := v1.Parse()
			s.update(BID, s.bidPrice2Element, s.depthBidList, &PriceLevel{
				Price:         v1.Price,
				Quantity:      v1.Quantity,
				PriceFloat:    price,
				QuantityFloat: quantity,
			})
		}
	} else if v, ok := event.(*futures.WsDepthEvent); ok {
		if v.LastUpdateID < s.LastUpdateID {
			return
		} else if v.PrevLastUpdateID != s.LastUpdateID && v.FirstUpdateID > s.LastUpdateID {
			log.Errorf(
				"%s.last update id error!FirstUpdateID:%s,LastUpdateID:%s,PrevLastUpdateID:%s,LocalUpdateID:%s",
				s.si, v.FirstUpdateID, v.LastUpdateID, v.PrevLastUpdateID, s.LastUpdateID,
			)
			s.Reset()
			return
		}

		s.LastUpdateID = v.LastUpdateID
		s.Time = int64(time.Now().UnixNano() / 1e6)
		for _, v1 := range v.Asks {
			price, quantity, _ := v1.Parse()
			s.update(ASK, s.askPrice2Element, s.depthAskList, &PriceLevel{
				Price:         v1.Price,
				Quantity:      v1.Quantity,
				PriceFloat:    price,
				QuantityFloat: quantity,
			})
		}
		for _, v1 := range v.Bids {
			price, quantity, _ := v1.Parse()
			s.update(BID, s.bidPrice2Element, s.depthBidList, &PriceLevel{
				Price:         v1.Price,
				Quantity:      v1.Quantity,
				PriceFloat:    price,
				QuantityFloat: quantity,
			})
		}
	}

	s.convertDepth()
}

func (s *DepthSrv) convertDepth() {
	for s.depthBidList.Len() > 1000 {
		s.depthBidList.Remove(s.depthBidList.Back())
	}
	for s.depthAskList.Len() > 1000 {
		s.depthAskList.Remove(s.depthAskList.Back())
	}

	depth := &Depth{
		LastUpdateID: s.LastUpdateID,
		Time:         s.Time,
		TradeTime:    0,
		Bids:         make([]PriceLevel, s.depthBidList.Len()),
		Asks:         make([]PriceLevel, s.depthAskList.Len()),
	}

	for ele, i := s.depthAskList.Front(), 0; ele != nil; ele = ele.Next() {
		depth.Asks[i] = *ele.Value.(*PriceLevel)
		i++
	}
	for ele, i := s.depthBidList.Front(), 0; ele != nil; ele = ele.Next() {
		depth.Bids[i] = *ele.Value.(*PriceLevel)
		i++
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	s.depth = depth
}

func (s *DepthSrv) GetDepth() *Depth {
	<-s.initCtx.Done()
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.depth
}
