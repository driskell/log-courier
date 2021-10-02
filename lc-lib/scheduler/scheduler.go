package scheduler

import (
	"container/heap"
	"time"
)

// Scheduler holds a list of scheduled objects and fires a timer when the next item is due
type Scheduler struct {
	tq          *timerQueue
	index       map[interface{}]*timerItem
	timer       *time.Timer
	next        time.Time
	isScheduled bool
}

// NewScheduler returns a new timer queue
func NewScheduler() *Scheduler {
	s := &Scheduler{
		tq:          new(timerQueue),
		index:       make(map[interface{}]*timerItem),
		timer:       time.NewTimer(0),
		isScheduled: true,
	}
	s.Reschedule(false)
	return s
}

// Set a new scheduled item in the queue to fire after the specified duration
// Must call Reschedule after to update the timer
func (s *Scheduler) Set(v interface{}, d time.Duration) {
	if item, ok := s.index[v]; ok {
		item.when = time.Now().Add(d)
		heap.Fix(s.tq, item.index)
	} else {
		item := &timerItem{
			value: v,
			when:  time.Now().Add(d),
		}
		s.index[v] = item
		heap.Push(s.tq, item)
	}

	if s.next == (*s.tq)[0].when {
		return
	}
}

// Remove a scheduled item from the scheduler
// If the item is not scheduled, this is a no-op
// Must call Reschedule after to update the timer
func (s *Scheduler) Remove(v interface{}) {
	if item, ok := s.index[v]; ok {
		heap.Remove(s.tq, item.index)
		delete(s.index, v)
	}
}

// Next returns the next item that is due, or nil if none are due
func (s *Scheduler) Next() interface{} {
	if len(*s.tq) == 0 || (*s.tq)[0].when.After(time.Now()) {
		return nil
	}
	item := heap.Pop(s.tq).(*timerItem)
	delete(s.index, item.value)
	return item.value
}

// OnNext returns a channel that will receive the current time when the next item is due
// Next is not guaranteed to return if OnNext fires so its result could still be nil
// Returns a nil channel if no items are scheduled, which can still be used in a switch but will never fire
func (s *Scheduler) OnNext() <-chan time.Time {
	if len(*s.tq) == 0 {
		return nil
	}
	return s.timer.C
}

// Reschedule arranges the timer again
func (s *Scheduler) Reschedule(didReceive bool) {
	if s.isScheduled && !didReceive && !s.timer.Stop() {
		<-s.timer.C
	}
	s.isScheduled = (len(*s.tq) != 0)
	if s.isScheduled {
		s.timer.Reset((*s.tq)[0].when.Sub(time.Now()))
	}
}
