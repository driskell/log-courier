package scheduler

import (
	"container/heap"
	"testing"
	"time"
)

func TestHeapOrder(t *testing.T) {
	tq := new(timerQueue)
	item1 := &timerItem{
		value: 1,
		when:  time.Now().Add(100 * time.Second),
	}
	item2 := &timerItem{
		value: 3,
		when:  time.Now().Add(300 * time.Second),
	}
	item3 := &timerItem{
		value: 2,
		when:  time.Now().Add(200 * time.Second),
	}
	heap.Push(tq, item1)
	heap.Push(tq, item2)
	heap.Push(tq, item3)
	item := heap.Pop(tq).(*timerItem)
	if item.value != 1 {
		t.Error("Unexpected scheduler ordering")
	}
	item = heap.Pop(tq).(*timerItem)
	if item.value != 2 {
		t.Error("Unexpected scheduler ordering")
	}
	heap.Push(tq, item3)
	item = heap.Pop(tq).(*timerItem)
	if item.value != 2 {
		t.Error("Unexpected scheduler ordering")
	}
	item = heap.Pop(tq).(*timerItem)
	if item.value != 3 {
		t.Error("Unexpected scheduler ordering")
	}
}

func TestHeapSwap(t *testing.T) {
	tq := new(timerQueue)
	item1 := &timerItem{
		value: 1,
		when:  time.Now().Add(100 * time.Second),
	}
	item2 := &timerItem{
		value: 3,
		when:  time.Now().Add(300 * time.Second),
	}
	item3 := &timerItem{
		value: 2,
		when:  time.Now().Add(200 * time.Second),
	}
	heap.Push(tq, item1)
	heap.Push(tq, item2)
	heap.Push(tq, item3)
	item := heap.Pop(tq).(*timerItem)
	if item.value != 1 {
		t.Error("Unexpected scheduler ordering")
	}
	item2.when = time.Now().Add(1 * time.Second)
	heap.Fix(tq, item2.index)
	item = heap.Pop(tq).(*timerItem)
	if item.value != 3 {
		t.Error("Unexpected scheduler ordering")
	}

}
