// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEvent_URL(t *testing.T) {
	const srvURL = "https://test.musicbrainz.org"
	for _, tc := range []struct{ mbid, want string }{
		{"", srvURL + "/event/create"},
		{"d98928e8-6757-4196-a945-e7145d94d9e4", srvURL + "/event/d98928e8-6757-4196-a945-e7145d94d9e4/edit"},
	} {
		rel := Event{MBID: tc.mbid}
		if got := rel.URL(srvURL); got != tc.want {
			t.Errorf("MBID %q yielded URL %q; want %q", tc.mbid, got, tc.want)
		}
	}
}

func TestEvent_Params(t *testing.T) {
	event := Event{
		Name:           "A New Event",
		Disambiguation: "for testing",
		Type:           EventType_Festival,
		Cancelled:      true,
		Setlist:        "* First Song\n* Second Song",
		BeginDate:      Date{2003, 12, 1},
		EndDate:        Date{2005, 4, 23},
		Time:           "20:15",
		Relationships: []Relationship{{
			Target: "0f50beab-d77d-4f0f-ac26-0b87d3e9b11b",
			Type:   LinkType_MainPerformer_Artist_Event,
		}},
		URLs: []URL{{
			URL:      "https://example.org/poster.jpg",
			LinkType: LinkType_Poster_Event_URL,
		}},
		EditNote: "here's the edit note",
	}

	rel := event.Relationships[0]
	want := url.Values{
		"edit-event.name":                    {event.Name},
		"edit-event.comment":                 {event.Disambiguation},
		"edit-event.type_id":                 {strconv.Itoa(int(event.Type))},
		"edit-event.cancelled":               {"1"},
		"edit-event.setlist":                 {event.Setlist},
		"edit-event.period.begin_date.year":  {strconv.Itoa(event.BeginDate.Year)},
		"edit-event.period.begin_date.month": {strconv.Itoa(event.BeginDate.Month)},
		"edit-event.period.begin_date.day":   {strconv.Itoa(event.BeginDate.Day)},
		"edit-event.period.end_date.year":    {strconv.Itoa(event.EndDate.Year)},
		"edit-event.period.end_date.month":   {strconv.Itoa(event.EndDate.Month)},
		"edit-event.period.end_date.day":     {strconv.Itoa(event.EndDate.Day)},
		"edit-event.time":                    {event.Time},
		"rels.0.target":                      {rel.Target},
		"rels.0.type":                        {strconv.Itoa(int(rel.Type))},
		"edit-event.url.0.text":              {event.URLs[0].URL},
		"edit-event.url.0.link_type_id":      {strconv.Itoa(int(event.URLs[0].LinkType))},
		"edit-event.edit_note":               {event.EditNote},
	}
	if diff := cmp.Diff(want, event.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}
}
