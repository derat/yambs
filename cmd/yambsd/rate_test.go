// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"testing"
	"time"
)

func TestRateMap(t *testing.T) {
	const (
		addr1 = "127.0.0.1"
		addr2 = "10.0.0.5"
		addr3 = "192.168.0.1"
	)
	start := time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)
	rm := newRateMap(3*time.Second, 2 /* size */)

	for _, tc := range []struct {
		addr string
		sec  int // seconds past start
		want bool
	}{
		{addr1, 0, true},
		{addr1, 0, false},
		{addr1, 1, false},
		{addr1, 2, false},
		{addr2, 2, true},
		{addr2, 2, false},
		{addr1, 3, true},
		{addr2, 3, false},
		{addr2, 5, true},
		{addr1, 5, false},
		{addr1, 6, true},
		{addr1, 8, false},
		{addr1, 9, true},
		{addr2, 9, true},
		{addr3, 9, true}, // evicts addr1
		{addr1, 9, true}, // now allowed
	} {
		now := start.Add(time.Duration(tc.sec) * time.Second)
		if got := rm.attempt(tc.addr, now); got != tc.want {
			t.Errorf("attempt(%q, %ds) = %v; want %v",
				tc.addr, tc.sec, got, tc.want)
		}
	}
}
