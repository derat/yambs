// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"testing"
)

func TestDetectScriptLocal(t *testing.T) {
	for _, tc := range []struct {
		titles []string
		want   string
	}{
		{[]string{"Latin chars", "More latin, with some punctuation!"}, "Latn"},
		// https://musicbrainz.org/release/5193d964-d732-4d81-a99f-37f8e5bb14bb
		{[]string{"Bā dù kōngjiān", "Bàn shòu rén"}, "Latn"},
		// https://musicbrainz.org/release/649c1359-9b17-3f4a-b2c7-912da50bf119
		// TODO: The MB release uses "Hant", but detecting traditional vs. simplified is hard.
		{[]string{"八度空間", "半獸人"}, "Hani"},
		// https://musicbrainz.org/release/c4716917-5064-48a8-a63b-00d024f57743
		{[]string{"화양연화", "잡아줘", "고엽"}, "Kore"},
		// https://musicbrainz.org/release/a24e9f34-d523-4671-8b1f-002c971b67cc
		{[]string{"初恋", "あなた", "誓い"}, "Jpan"},
		// https://musicbrainz.org/release/52bd1b33-f4bd-4e6d-894f-d62592554808
		{[]string{"αριθμός τέσσερα"}, "Grek"},
		{[]string{"αριθμός τέσσερα", "जलाना  (Jalaana)", "Nammu"}, ""},
	} {
		if got := detectScriptLocal(tc.titles); got != tc.want {
			t.Errorf("detectScriptLocal(%q) = %q; want %q", tc.titles, got, tc.want)
		}
	}
}
