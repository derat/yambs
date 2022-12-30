// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	for _, tc := range []struct {
		orig   string
		max    int
		ellide bool
		want   string
	}{
		{"abc", 4, true, "abc"},
		{"abc", 3, true, "abc"},
		{"abc", 2, true, "a…"},
		{"abc", 1, true, "…"},
		{"abc", 4, false, "abc"},
		{"abc", 3, false, "abc"},
		{"abc", 2, false, "ab"},
		{"abc", 1, false, "a"},
	} {
		if got := truncate(tc.orig, tc.max, tc.ellide); got != tc.want {
			t.Errorf("truncate(%q, %v, %v) = %q; want %q", tc.orig, tc.max, tc.ellide, got, tc.want)
		}
	}
}
