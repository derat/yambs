// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"container/list"
	"sync"
	"time"
)

// rateMap performs basic rate-limiting by string key (e.g. client IP).
// It can be used concurrently from multiple goroutines.
type rateMap struct {
	m  map[string]*rateEntry // indexed by key
	ls list.List             // contains keys, oldest in front
	mu sync.Mutex            // protects m and ls
}

func newRateMap() *rateMap { return &rateMap{m: make(map[string]*rateEntry)} }

type rateEntry struct {
	el *list.Element // element in rateMap.ls
	tm time.Time     // time of last successful update
}

// update handles a request by client k at time now.
// If this is k's first request or the last request was dur or longer ago,
// the stored request time is updated and true is returned.
// Otherwise (i.e. k's last request was too recent), the stored request time
// is not updated and false is returned.
func (rm *rateMap) update(k string, now time.Time, dur time.Duration) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var allowed bool
	if re, ok := rm.m[k]; ok {
		// Only update the existing entry if the request is allowed.
		if allowed = now.Sub(re.tm) >= dur; allowed {
			re.tm = now
			rm.ls.MoveToBack(re.el)
		}
	} else {
		// Otherwise, allow the request and add a new entry.
		allowed = true
		rm.m[k] = &rateEntry{rm.ls.PushBack(k), now}
	}

	// Delete any existing entries that are dur or older.
	for rm.ls.Len() > 0 {
		el := rm.ls.Front()
		k := el.Value.(string)
		if now.Sub(rm.m[k].tm) < dur {
			break
		}
		delete(rm.m, k)
		rm.ls.Remove(el)
	}

	return allowed
}
