// Copyright 2022 Daniel Erat.
// All rights reserved.

package cache

import (
	"runtime"
	"testing"
)

func TestLRU(t *testing.T) {
	lru := NewLRU(2)

	// ln returns the caller's caller's line number.
	ln := func(skip int) int {
		_, _, line, _ := runtime.Caller(skip + 1)
		return line
	}

	// Gross convenience functions for calling methods and checking results.
	set := func(key, val string) { lru.Set(key, val) }
	get := func(key string, wantVal interface{}, wantOK bool) {
		if val, ok := lru.Get(key); val != wantVal || ok != wantOK {
			t.Errorf("L%d: Get(%q) = %q, %v; want %q, %v", ln(1), key, val, ok, wantVal, wantOK)
		}
	}
	testAndSet := func(key string, val interface{}, test func(string) bool, want bool) {
		fn := func(old interface{}) bool { return test(old.(string)) }
		if ok := lru.TestAndSet(key, val, fn); ok != want {
			t.Errorf("L%d: TestAndSet(%q, %q, ...) = %v; want %v", ln(1), key, val, ok, want)
		}
	}

	const (
		k1 = "k1"
		k2 = "k2"
		k3 = "k3"
	)

	// Set and update a key.
	get(k1, nil, false)
	set(k1, "foo")
	get(k1, "foo", true)
	set(k1, "bar")
	get(k1, "bar", true)

	// Set a second key and check that the first is still there.
	get(k2, nil, false)
	set(k2, "yams")
	get(k2, "yams", true)
	get(k1, "bar", true)

	// Set a third key, which should evict the second key since it was accessed the longest ago.
	set(k3, "trout")
	get(k1, "bar", true)
	get(k2, nil, false)
	get(k3, "trout", true)

	// Check that TestAndSet only sets when the test function returns true.
	testAndSet(k1, "baz", func(v string) bool { return v == "bogus" }, false)
	get(k1, "bar", true)
	testAndSet(k1, "baz", func(v string) bool { return v == "bar" }, true)
	get(k1, "baz", true)

	// TestAndSet should set without calling the test function when the key isn't present.
	testAndSet(k2, "apples", func(v string) bool {
		t.Error("called unexpectedly")
		return false
	}, true)
	get(k2, "apples", true)
	get(k3, nil, false) // should've been evicted
}
