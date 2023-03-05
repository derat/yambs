// Copyright 2023 Daniel Erat.
// All rights reserved.

package strutil

import (
	"testing"
)

func TestLevenshtein(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want Edits
	}{
		{"a", "", Edits{Dels: 1}},
		{"", "a", Edits{Ins: 1}},
		{"a", "b", Edits{Subs: 1}},
		{"ab", "b", Edits{Dels: 1}},
		{"a", "ab", Edits{Ins: 1}},
		{"abde", "bcd", Edits{Ins: 1, Dels: 2}},
		{"my name is john", "my first name is john", Edits{Ins: 6}},
		{"a red cow", "a tan cow", Edits{Subs: 3}},
	} {
		if got := Levenshtein(tc.a, tc.b); got != tc.want {
			t.Errorf("Levenshtein(%q, %q) = %+v; want %+v", tc.a, tc.b, got, tc.want)
		}
	}
}
