// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"2022-05-04", "2022-05-04"},
		{"2022-05", "2022-05-01"},
		{"2022", "2022-01-01"},
	} {
		if d, err := parseDate(tc.in); err != nil {
			t.Errorf("parseDate(%q) failed: %v", tc.in, err)
		} else if got := d.Format("2006-01-02"); got != tc.want {
			t.Errorf("parseDate(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseDuration(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want time.Duration
	}{
		{"0", 0 * time.Millisecond},
		{"1", 1 * time.Millisecond},
		{"1000", time.Second},
		{"125231.98", 125*time.Second + 231*time.Millisecond + 980*time.Microsecond},
		{":45", 45 * time.Second},
		{"3:45", 3*time.Minute + 45*time.Second},
		{"0:23.678", 23*time.Second + 678*time.Millisecond},
		{"1:23:45", time.Hour + 23*time.Minute + 45*time.Second},
	} {
		if got, err := parseDuration(tc.in); err != nil {
			t.Errorf("parseDuration(%q) failed: %v", tc.in, err)
		} else if got != tc.want {
			t.Errorf("parseDuration(%q) = %s; want %s", tc.in, got, tc.want)
		}
	}
}
