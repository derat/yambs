// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"time"

	"github.com/derat/yambs/cache"
)

// rateMap is a wrapper around cache.LRU that simplifies rate-limiting requests.
type rateMap struct {
	lru   *cache.LRU
	delay time.Duration // min time between requests
}

// newRateMap returns a new rateMap that makes each client wait delay between requests.
// size is the maximum size of the underlying cache.
func newRateMap(delay time.Duration, size int) *rateMap {
	return &rateMap{cache.NewLRU(size), delay}
}

// attempt should be called in response to a request from key at now.
// It returns true iff the client's last request was rm.delay or more in the past.
func (rm *rateMap) attempt(key string, now time.Time) bool {
	return rm.lru.TestAndSet(key, now, func(last interface{}) bool {
		return now.Sub(last.(time.Time)) >= rm.delay
	})
}
