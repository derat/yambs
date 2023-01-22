// Copyright 2022 Daniel Erat.
// All rights reserved.

package main

import (
	"reflect"
	"testing"
)

func TestWrap(t *testing.T) {
	for _, tc := range []struct {
		orig string
		max  int
		want []string
	}{
		{"abcdef", 5, []string{"abcdef"}},
		{"abcdef", 6, []string{"abcdef"}},
		{"abcdef", 7, []string{"abcdef"}},
		{"abc def", 2, []string{"abc", "def"}},
		{"abc def", 3, []string{"abc", "def"}},
		{"abc def", 4, []string{"abc", "def"}},
		{"abc def", 5, []string{"abc", "def"}},
		{"abc def", 6, []string{"abc", "def"}},
		{"abc def", 7, []string{"abc def"}},
		{"abc   def", 2, []string{"abc", "def"}},
		{"abc   def", 3, []string{"abc", "def"}},
		{"abc   def", 4, []string{"abc", "def"}},
		{"abc   def", 8, []string{"abc", "def"}},
		{"abc   def", 9, []string{"abc   def"}},
		{"abc\ndef ghi", 3, []string{"abc", "def", "ghi"}},
	} {
		if got := wrap(tc.orig, tc.max); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("wrap(%q, %d) = %q; want %q", tc.orig, tc.max, got, tc.want)
		}
	}
}
