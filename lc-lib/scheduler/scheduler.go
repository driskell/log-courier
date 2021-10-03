package scheduler

import (
	"container/heap"
	"time"
)

// Callback is a callback that can be scheduled
type Callback func()

// Scheduler holds a list of scheduled objects and fires a timer when the next item is due
type Scheduler struct {
	tq    *timerQueue
	index map[interface{}]*timerItem
	timer *time.Timer
	next  *time.Time
}

// NewScheduler returns a new timer queue
func NewScheduler() *Scheduler {
	s := &Scheduler{
		tq:    new(timerQueue),
		index: make(map[interface{}]*timerItem),
		timer: time.NewTimer(0),
	}
	s.Reschedule()
	return s
}

// Set a new scheduled item in the queue to return on Next() after the specified duration
func (s *Scheduler) Set(v interface{}, d time.Duration) {
	s.set(v, d, nil)
}

// SetCallback sets a callback that is automatically called during Next() after the specified duration
// Next() will not return the item unlike Set()
func (s *Scheduler) SetCallback(v interface{}, d time.Duration, callback Callback) {
	s.set(v, d, callback)
}

// set manages updating the schedule for an internal timerItem
func (s *Scheduler) set(v interface{}, d time.Duration, callback Callback) {
	if item, ok := s.index[v]; ok {
		item.when = time.Now().Add(d)
		heap.Fix(s.tq, item.index)
	} else {
		item := &timerItem{
			value:    v,
			callback: callback,
			when:     time.Now().Add(d),
		}
		s.index[v] = item
		heap.Push(s.tq, item)
	}

	if s.next != nil && ((*s.tq)[0].when.After(*s.next) || *s.next == (*s.tq)[0].when) {
		return
	}

	// Update timer since the new one is earlier - we optimise for the case
	// where multiple Set are unlikely to need rescheduling as the earliest
	// callback will occur first and trigger only a single reschedule
	s.Reschedule()
}

// Remove a scheduled item or callback from the scheduler
// If the item is not scheduled, this is a no-op
func (s *Scheduler) Remove(v interface{}) {
	if item, ok := s.index[v]; ok {
		heap.Remove(s.tq, item.index)
		delete(s.index, v)
	}
}

// Next returns the next item that is due, or nil if none are due
// For callback items it will handle them silently, so may return nil even though callbacks were called
func (s *Scheduler) Next() interface{} {
	// Handle all available items
	for {
		if len(*s.tq) == 0 || (*s.tq)[0].when.After(time.Now()) {
			return nil
		}
		item := heap.Pop(s.tq).(*timerItem)
		delete(s.index, item.value)
		// If not a callback - return it
		if item.callback == nil {
			return item.value
		}
		item.callback()
	}
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
func (s *Scheduler) Reschedule() {
	if !s.timer.Stop() && len(s.timer.C) != 0 {
		<-s.timer.C
	}
	if len(*s.tq) != 0 {
		s.next = &(*s.tq)[0].when
		s.timer.Reset(time.Until(*s.next))
	} else {
		s.next = nil
	}
}
