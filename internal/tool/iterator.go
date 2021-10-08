package tool

import "time"

type DelayIterator struct {
	index     int
	delayList []time.Duration
}

func NewDelayIterator() *DelayIterator {
	return &DelayIterator{
		delayList: []time.Duration{
			0 * time.Millisecond,
			10 * time.Millisecond,
			10 * time.Millisecond,
			10 * time.Millisecond,
			1000 * time.Millisecond,
			2000 * time.Millisecond,
			5000 * time.Millisecond,
			10000 * time.Millisecond,
			15000 * time.Millisecond,
			30000 * time.Millisecond,
			60000 * time.Millisecond,
		},
	}
}

func (s *DelayIterator) SetDelayList(delayList []time.Duration) {
	s.delayList = delayList
}

func (s *DelayIterator) Reset() {
	s.index = 0
}

func (s *DelayIterator) Delay() {
	if s.index >= len(s.delayList) {
		time.Sleep(s.delayList[len(s.delayList)-1])
	} else {
		time.Sleep(s.delayList[s.index])
		s.index++
	}
}
