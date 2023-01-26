// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLabel_URL(t *testing.T) {
	const srv = "test.musicbrainz.org"
	for _, tc := range []struct{ mbid, want string }{
		{"", "https://" + srv + "/label/create"},
		{"d98928e8-6757-4196-a945-e7145d94d9e4", "https://" + srv + "/label/d98928e8-6757-4196-a945-e7145d94d9e4/edit"},
	} {
		rel := Label{MBID: tc.mbid}
		if got := rel.URL(srv); got != tc.want {
			t.Errorf("MBID %q yielded URL %q; want %q", tc.mbid, got, tc.want)
		}
	}
}

func TestLabel_Params(t *testing.T) {
	label := Label{
		Name:           "A New Label",
		Disambiguation: "for testing",
		Type:           LabelType_Publisher,
		AreaName:       "France",
		LabelCode:      "12345",
		IPICodes:       []string{"123", "456"},
		ISNICodes:      []string{"abc", "def"},
		BeginYear:      2003,
		BeginMonth:     12,
		BeginDay:       1,
		EndYear:        2005,
		EndMonth:       4,
		EndDay:         23,
		Ended:          true,
		Relationships: []Relationship{{
			Target: "0f50beab-d77d-4f0f-ac26-0b87d3e9b11b",
			Type:   LinkType_ArrangedFor_Label_Release,
		}},
		URLs: []URL{{
			URL:      "https://label.bandcamp.com/",
			LinkType: LinkType_Bandcamp_Label_URL,
		}},
		EditNote: "here's the edit note",
	}

	rel := label.Relationships[0]
	want := url.Values{
		"edit-label.name":                    {label.Name},
		"edit-label.comment":                 {label.Disambiguation},
		"edit-label.type_id":                 {strconv.Itoa(int(label.Type))},
		"edit-label.area.name":               {label.AreaName},
		"edit-label.label_code":              {label.LabelCode},
		"edit-label.ipi_codes.0":             {label.IPICodes[0]},
		"edit-label.ipi_codes.1":             {label.IPICodes[1]},
		"edit-label.isni_codes.0":            {label.ISNICodes[0]},
		"edit-label.isni_codes.1":            {label.ISNICodes[1]},
		"edit-label.period.begin_date.year":  {strconv.Itoa(label.BeginYear)},
		"edit-label.period.begin_date.month": {strconv.Itoa(label.BeginMonth)},
		"edit-label.period.begin_date.day":   {strconv.Itoa(label.BeginDay)},
		"edit-label.period.end_date.year":    {strconv.Itoa(label.EndYear)},
		"edit-label.period.end_date.month":   {strconv.Itoa(label.EndMonth)},
		"edit-label.period.end_date.day":     {strconv.Itoa(label.EndDay)},
		"edit-label.period.ended":            {"1"},
		"rels.0.target":                      {rel.Target},
		"rels.0.type":                        {strconv.Itoa(int(rel.Type))},
		"edit-label.url.0.text":              {label.URLs[0].URL},
		"edit-label.url.0.link_type_id":      {strconv.Itoa(int(label.URLs[0].LinkType))},
		"edit-label.edit_note":               {label.EditNote},
	}
	if diff := cmp.Diff(want, label.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}
}
