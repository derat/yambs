// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestArtist_URL(t *testing.T) {
	const srv = "test.musicbrainz.org"
	for _, tc := range []struct{ mbid, want string }{
		{"", "https://" + srv + "/artist/create"},
		{"d98928e8-6757-4196-a945-e7145d94d9e4", "https://" + srv + "/artist/d98928e8-6757-4196-a945-e7145d94d9e4/edit"},
	} {
		rel := Artist{MBID: tc.mbid}
		if got := rel.URL(srv); got != tc.want {
			t.Errorf("MBID %q yielded URL %q; want %q", tc.mbid, got, tc.want)
		}
	}
}

func TestArtist_Params(t *testing.T) {
	artist := Artist{
		Name:           "Mr. Mixer",
		SortName:       "Mixer, Mr.",
		Disambiguation: "for testing",
		Type:           ArtistType_Person,
		AreaName:       "France",
		IPICodes:       []string{"123", "456"},
		ISNICodes:      []string{"abc", "def"},
		BeginAreaName:  "San Francisco",
		BeginDate:      Date{1973, 12, 1},
		EndAreaName:    "Oakland",
		EndDate:        Date{2005, 4, 23},
		Ended:          true,
		Relationships: []Relationship{{
			Target: "0f50beab-d77d-4f0f-ac26-0b87d3e9b11b",
			Type:   LinkType_ArrangedFor_Label_Release,
		}},
		URLs: []URL{{
			URL:      "https://artist.bandcamp.com/",
			LinkType: LinkType_Bandcamp_Artist_URL,
		}},
		EditNote: "here's the edit note",
	}

	rel := artist.Relationships[0]
	want := url.Values{
		"edit-artist.name":                    {artist.Name},
		"edit-artist.sort_name":               {artist.SortName},
		"edit-artist.comment":                 {artist.Disambiguation},
		"edit-artist.type_id":                 {strconv.Itoa(int(artist.Type))},
		"edit-artist.area.name":               {artist.AreaName},
		"edit-artist.ipi_codes.0":             {artist.IPICodes[0]},
		"edit-artist.ipi_codes.1":             {artist.IPICodes[1]},
		"edit-artist.isni_codes.0":            {artist.ISNICodes[0]},
		"edit-artist.isni_codes.1":            {artist.ISNICodes[1]},
		"edit-artist.begin_area.name":         {artist.BeginAreaName},
		"edit-artist.period.begin_date.year":  {strconv.Itoa(artist.BeginDate.Year)},
		"edit-artist.period.begin_date.month": {strconv.Itoa(artist.BeginDate.Month)},
		"edit-artist.period.begin_date.day":   {strconv.Itoa(artist.BeginDate.Day)},
		"edit-artist.end_area.name":           {artist.EndAreaName},
		"edit-artist.period.end_date.year":    {strconv.Itoa(artist.EndDate.Year)},
		"edit-artist.period.end_date.month":   {strconv.Itoa(artist.EndDate.Month)},
		"edit-artist.period.end_date.day":     {strconv.Itoa(artist.EndDate.Day)},
		"edit-artist.period.ended":            {"1"},
		"rels.0.target":                       {rel.Target},
		"rels.0.type":                         {strconv.Itoa(int(rel.Type))},
		"edit-artist.url.0.text":              {artist.URLs[0].URL},
		"edit-artist.url.0.link_type_id":      {strconv.Itoa(int(artist.URLs[0].LinkType))},
		"edit-artist.edit_note":               {artist.EditNote},
	}
	if diff := cmp.Diff(want, artist.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}
}
