// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestRecording_Params(t *testing.T) {
	// This is sort of a "make sure I'm capable of typing the same thing twice" test,
	// but I guess it's useful for verifying that repeated fields are properly translated
	// to query parameters.
	rec := Recording{
		Name:   "Creating Cyclical Headaches",
		Artist: "Prefuse 73",
		Artists: []ArtistCredit{
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
		ISRCs:  []string{"USS1Z9900001", "AA6Q72000047"},
		URLs: []URL{
			{URL: "https://www.example.org/foo", LinkType: LinkType_Crowdfunding_Recording_URL},
			{URL: "https://www.example.org/bar", LinkType: LinkType_DownloadForFree_Recording_URL},
		},
		EditNote: "here's the edit note",
	}

	want := url.Values{
		"artist": {rec.Artist},
		"edit-recording.artist_credit.names.0.artist.id":   {strconv.Itoa(int(rec.Artists[0].ID))},
		"edit-recording.artist_credit.names.0.join_phrase": {rec.Artists[0].JoinPhrase},
		"edit-recording.artist_credit.names.0.name":        {rec.Artists[0].NameAsCredited},
		"edit-recording.artist_credit.names.1.artist.name": {rec.Artists[1].Name},
		"edit-recording.artist_credit.names.1.name":        {rec.Artists[1].NameAsCredited},
		"edit-recording.edit_note":                         {rec.EditNote},
		"edit-recording.isrcs.0":                           {rec.ISRCs[0]},
		"edit-recording.isrcs.1":                           {rec.ISRCs[1]},
		"edit-recording.length":                            {strconv.Itoa(int(rec.Length / time.Millisecond))},
		"edit-recording.name":                              {rec.Name},
		"edit-recording.url.0.text":                        {rec.URLs[0].URL},
		"edit-recording.url.0.link_type_id":                {strconv.Itoa(int(rec.URLs[0].LinkType))},
		"edit-recording.url.1.text":                        {rec.URLs[1].URL},
		"edit-recording.url.1.link_type_id":                {strconv.Itoa(int(rec.URLs[1].LinkType))},
	}
	if diff := cmp.Diff(want, rec.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}
}
