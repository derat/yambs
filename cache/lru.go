// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package cache contains cache implementations.
package cache

import (
	"container/list"
	"sync"
)

// LRU implements a fixed-size LRU cache with string keys.
// It can be used concurrently from multiple goroutines.
type LRU struct {
	m   map[string]*lruEntry // indexed by key
	ls  list.List            // contains keys, oldest in front
	mu  sync.Mutex           // protects m and ls
	max int                  // maximum items to store
}

// NewLRU returns a new LRU that will hold up to max items.
func NewLRU(max int) *LRU { return &LRU{m: make(map[string]*lruEntry), max: max} }

type lruEntry struct {
	el  *list.Element // element in LRU.ls
	val interface{}   // value associated with key
}

// Get returns the value associated with key.
// If the key isn't present in the map, nil and false are returned.
func (lru *LRU) Get(key string) (val interface{}, ok bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	ent, ok := lru.m[key]
	if !ok {
		return nil, false
	}
	lru.ls.MoveToBack(ent.el)
	return ent.val, true
}

// Set saves a mapping from key to val.
func (lru *LRU) Set(key string, val interface{}) {
	if added := lru.TestAndSet(key, val, nil); !added {
		// Perform an assertion.
		panic("TestAndSet didn't save value when passed nil test function")
	}
}

// TestAndSet saves a mapping from key to val and returns true if key isn't
// already present or if test returns true when passed the existing value.
// If test returns false, the mapping is not saved and false is returned.
// The new mapping is set unconditionally (and true is returned) if test is nil.
func (lru *LRU) TestAndSet(key string, val interface{}, test func(interface{}) bool) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	// This is dumb, but whatever.
	if lru.max == 0 {
		return true
	}

	// If the key is already present in the map and the test function returns true,
	// update the value and move the entry to the end of the expiry list.
	if ent, ok := lru.m[key]; ok {
		if test != nil && !test(ent.val) {
			return false
		}
		ent.val = val
		lru.ls.MoveToBack(ent.el)
		return true
	}

	// Shrink the cache down to just below the maximum size.
	for lru.ls.Len() >= lru.max {
		el := lru.ls.Front()
		delete(lru.m, el.Value.(string))
		lru.ls.Remove(el)
	}

	// Add the new entry.
	lru.m[key] = &lruEntry{lru.ls.PushBack(key), val}
	return true
}
