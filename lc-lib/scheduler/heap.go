package scheduler

import "time"

// timerItem holds an entry in the queue
type timerItem struct {
	value    interface{}
	callback Callback
	when     time.Time
	index    int
}

// A timerQueue implements heap.Interface and holds timerItems.
type timerQueue []*timerItem

// Len is part of heap.Interface
func (tq timerQueue) Len() int { return len(tq) }

// Less is part of heap.Interface
func (tq timerQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return tq[i].when.Before(tq[j].when)
}

// Swap is part of heap.Interface
func (tq timerQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index = i
	tq[j].index = j
}

// Push is part of heap.Interface
func (tq *timerQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*timerItem)
	item.index = n
	*tq = append(*tq, item)
}

// Pop is part of heap.Interface
func (tq *timerQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*tq = old[0 : n-1]
	return item
}
