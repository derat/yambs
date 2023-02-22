// Copyright 2023 Daniel Erat.
// All rights reserved.

package mbdb

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want edits
	}{
		{"a", "", edits{dels: 1}},
		{"", "a", edits{ins: 1}},
		{"a", "b", edits{subs: 1}},
		{"ab", "b", edits{dels: 1}},
		{"a", "ab", edits{ins: 1}},
		{"abde", "bcd", edits{ins: 1, dels: 2}},
		{"my name is john", "my first name is john", edits{ins: 6}},
		{"a red cow", "a tan cow", edits{subs: 3}},
	} {
		if got := levenshtein(tc.a, tc.b); got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %+v; want %+v", tc.a, tc.b, got, tc.want)
		}
	}
}
