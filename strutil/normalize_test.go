// Copyright 2023 Daniel Erat.
// All rights reserved.

package strutil

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", ""},
		{"abc", "abc"},
		{"‘Áç₉µ’", "‘Ac9μ’"},
	} {
		if got := Normalize(tc.in); got != tc.want {
			t.Errorf("Normalize(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
