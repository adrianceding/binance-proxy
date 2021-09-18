package tool

import "time"

type DelayIterator struct {
	index     int
	delayList []time.Duration
}

func NewDelayIterator() *DelayIterator {
	return &DelayIterator{
		delayList: []time.Duration{
			0 * time.Microsecond,
			10 * time.Microsecond,
			10 * time.Microsecond,
			10 * time.Microsecond,
			1000 * time.Microsecond,
			2000 * time.Microsecond,
			5000 * time.Microsecond,
			10000 * time.Microsecond,
			15000 * time.Microsecond,
			30000 * time.Microsecond,
			60000 * time.Microsecond,
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
	if s.index+1 > len(s.delayList) {
		time.Sleep(s.delayList[len(s.delayList)])
	} else {
		time.Sleep(s.delayList[s.index])
		s.index++
	}
}
