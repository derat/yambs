// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRecording_SetParams(t *testing.T) {
	rec := Recording{
		Title:  "Creating Cyclical Headaches",
		Artist: "Prefuse 73",
		ArtistCredits: []ArtistCredit{
			{
				ID:             56535,
				NameAsCredited: "Prefuse Seventy Three",
				JoinPhrase:     " feat. ",
			},
			{
				Name:           "Four Tet",
				NameAsCredited: "4 Tet",
			},
		},
		Length: 3*time.Minute + 27*time.Second + 400*time.Millisecond,
	}

	want := strings.Join([]string{
		"artist=" + url.QueryEscape(rec.Artist),
		"edit-recording.artist_credit.names.0.artist.id=" + fmt.Sprint(rec.ArtistCredits[0].ID),
		"edit-recording.artist_credit.names.0.join_phrase=" + url.QueryEscape(rec.ArtistCredits[0].JoinPhrase),
		"edit-recording.artist_credit.names.0.name=" + url.QueryEscape(rec.ArtistCredits[0].NameAsCredited),
		"edit-recording.artist_credit.names.1.artist.name=" + url.QueryEscape(rec.ArtistCredits[1].Name),
		"edit-recording.artist_credit.names.1.name=" + url.QueryEscape(rec.ArtistCredits[1].NameAsCredited),
		"edit-recording.length=" + fmt.Sprintf("%d", rec.Length/time.Millisecond),
		"edit-recording.name=" + url.QueryEscape(rec.Title),
	}, "&")

	vals := make(url.Values)
	rec.SetParams(vals)
	if got := vals.Encode(); got != want {
		t.Errorf("Incorrect query params for recording:\ngot  %q\nwant %q", got, want)
	}
}
