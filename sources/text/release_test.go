// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/google/go-cmp/cmp"
)

func TestReadEdits_Release_All(t *testing.T) {
	const input = `
title=Release Title
release_group=4b52bddc-0587-4bcf-9e05-5c9fca260a21
types=Single,Soundtrack
disambiguation=Not the same
annotation=This was actually released accidentally.
barcode=1234567890
language=eng
script=Latn
status=Official
packaging=Digipak
event0_date=2021-04-05
event0_country=xw
event1_year=2020
event1_month=1
event1_day=2
label0_mbid=bb9bba31-cb31-440d-a813-f5bf884f6adb
label0_catalog=CAT012
label1_name=Some Label
artist0_mbid=cd72c13c-a74e-4617-af5f-658409a36894
artist0_credited=First Artist
artist0_join= feat. 
artist1_name=Second Artist
medium0_format=CD
medium0_name=First Disc
medium0_track0_title=First Track
medium0_track0_number=1
medium0_track0_recording=c347b502-7ac8-46bb-a19a-a5e758900fe1
medium0_track0_length=1:02.56
medium0_track0_artist0_mbid=4d0db17e-14e0-4904-a39b-a9ffa81890df
medium0_track0_artist0_credited=Artist A
medium0_track0_artist0_join= & 
medium0_track0_artist1_name=Artist B
medium0_track1_title=Second Track
medium0_track1_length=45001
url0_url=https://www.example.org/a
url0_type=75
url1_url=https://www.example.org/b
edit_note=https://www.example.org/
`
	got, err := ReadEdits(context.Background(),
		strings.NewReader(strings.TrimLeft(input, "\n")),
		KeyVal, seed.ReleaseType, "", nil, db.NewDB(db.DisallowQueries))
	if err != nil {
		t.Fatal("ReadEdits failed:", err)
	}
	want := []seed.Edit{
		&seed.Release{
			Title:        "Release Title",
			ReleaseGroup: "4b52bddc-0587-4bcf-9e05-5c9fca260a21",
			Types: []seed.ReleaseGroupType{
				seed.ReleaseGroupType_Single,
				seed.ReleaseGroupType_Soundtrack,
			},
			Disambiguation: "Not the same",
			Annotation:     "This was actually released accidentally.",
			Barcode:        "1234567890",
			Language:       "eng",
			Script:         "Latn",
			Status:         seed.ReleaseStatus_Official,
			Packaging:      seed.ReleasePackaging_Digipak,
			Events: []seed.ReleaseEvent{
				{Year: 2021, Month: 4, Day: 5, Country: "xw"},
				{Year: 2020, Month: 1, Day: 2},
			},
			Labels: []seed.ReleaseLabel{
				{MBID: "bb9bba31-cb31-440d-a813-f5bf884f6adb", CatalogNumber: "CAT012"},
				{Name: "Some Label"},
			},
			Artists: []seed.ArtistCredit{
				{MBID: "cd72c13c-a74e-4617-af5f-658409a36894", NameAsCredited: "First Artist", JoinPhrase: " feat. "},
				{Name: "Second Artist"},
			},
			Mediums: []seed.Medium{{
				Format: seed.MediumFormat_CD,
				Name:   "First Disc",
				Tracks: []seed.Track{
					{
						Title:     "First Track",
						Number:    "1",
						Recording: "c347b502-7ac8-46bb-a19a-a5e758900fe1",
						Length:    time.Minute + 2*time.Second + 560*time.Millisecond,
						Artists: []seed.ArtistCredit{
							{MBID: "4d0db17e-14e0-4904-a39b-a9ffa81890df", NameAsCredited: "Artist A", JoinPhrase: " & "},
							{Name: "Artist B"},
						},
					},
					{Title: "Second Track", Length: 45001 * time.Millisecond},
				},
			}},
			URLs: []seed.URL{
				{URL: "https://www.example.org/a", LinkType: seed.LinkType_DownloadForFree_Release_URL},
				{URL: "https://www.example.org/b"},
			},
			EditNote: "https://www.example.org/",
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("ReadEdits returned wrong edits:\n" + diff)
	}
}
