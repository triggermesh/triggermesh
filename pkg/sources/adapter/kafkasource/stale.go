package kafkasource

import (
	"sync"
	"time"
)

type item struct {
	object interface{}
	added  time.Time
}

// StaleList is a list of items that timeout lazily,
// only checking for item expiration when a new one is added.
//
// This should not be used for storing a big number of items.
type StaleList struct {
	items   []item
	timeout time.Duration
	m       sync.Mutex
}

func NewStaleList(timeout time.Duration) *StaleList {
	return &StaleList{
		items:   []item{},
		timeout: timeout,
	}
}

func (sl *StaleList) count() int {
	index := -1
	for i := range sl.items {
		if time.Since(sl.items[i].added) > sl.timeout {
			index = i
			continue
		}
		break
	}

	if index != -1 {
		sl.items = sl.items[index+1:]
	}

	return len(sl.items)
}

// Adds new element to the list and updates the count, removing
// any stale items from it.
func (sl *StaleList) AddAndCount(object interface{}) int {
	sl.m.Lock()
	defer sl.m.Unlock()

	sl.items = append(sl.items, item{
		added:  time.Now(),
		object: object,
	})

	return sl.count()
}

// Updates the count removing any stale items from it.
func (sl *StaleList) Count() int {
	sl.m.Lock()
	defer sl.m.Unlock()

	return sl.count()
}
