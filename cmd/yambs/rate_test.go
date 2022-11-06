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
		dur   = 3 * time.Second
	)
	start := time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)
	rm := newRateMap()

	for _, tc := range []struct {
		addr string
		sec  int // seconds past start
		want bool
		size int
	}{
		{addr1, 0, true, 1},
		{addr1, 1, false, 1},
		{addr1, 2, false, 1},
		{addr2, 2, true, 2},
		{addr1, 3, true, 2},
		{addr2, 3, false, 2},
		{addr2, 5, true, 2},
		{addr1, 5, false, 2},
		{addr1, 6, true, 2},
		{addr1, 8, false, 1}, // addr2 expires
		{addr1, 9, true, 1},
	} {
		now := start.Add(time.Duration(tc.sec) * time.Second)
		if got := rm.update(tc.addr, now, dur); got != tc.want {
			t.Errorf("update(%q, %ds, %v) = %v; want %v",
				tc.addr, tc.sec, dur, got, tc.want)
		}
		if len(rm.m) != tc.size || rm.ls.Len() != tc.size {
			t.Errorf("update(%q, %ds, %v): map size is %d and list size is %d; want %d",
				tc.addr, tc.sec, dur, len(rm.m), rm.ls.Len(), tc.size)
		}
	}
}
