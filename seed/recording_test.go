// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"fmt"
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
		Disambiguation: "for testing",
		Length:         3*time.Minute + 27*time.Second + 400*time.Millisecond,
		ISRCs:          []string{"USS1Z9900001", "AA6Q72000047"},
		URLs: []URL{
			{URL: "https://www.example.org/foo", LinkType: LinkType_Crowdfunding_Recording_URL},
			{URL: "https://www.example.org/bar", LinkType: LinkType_DownloadForFree_Recording_URL},
		},
		Relationships: []Relationship{
			{
				Target:    "27aff659-aba5-41e5-8d35-9835fc9017d4",
				Type:      LinkType_Edit_Recording_Recording,
				BeginDate: Date{Year: 2020},
				EndDate:   Date{2022, 2, 15},
				Ended:     true,
				Backward:  true,
			},
			{
				TypeUUID: "93078fc7-6585-40a7-ab7f-6acb9da65b84",
				Attributes: []RelationshipAttribute{
					{
						Type:       LinkAttributeType_LeadVocals,
						CreditedAs: "some credit",
					},
					{
						TypeUUID:  "1d05bc4b-9884-4c9f-9f69-616f119047f3",
						TextValue: "some text",
					},
				},
			},
		},
		EditNote: "here's the edit note",
	}

	rel0 := rec.Relationships[0]
	beginDate := fmt.Sprintf("%04d", rel0.BeginDate.Year)
	endDate := fmt.Sprintf("%04d-%02d-%02d", rel0.EndDate.Year, rel0.EndDate.Month, rel0.EndDate.Day)
	rel1 := rec.Relationships[1]
	want := url.Values{
		"artist": {rec.Artist},
		"edit-recording.artist_credit.names.0.artist.id":   {strconv.Itoa(int(rec.Artists[0].ID))},
		"edit-recording.artist_credit.names.0.join_phrase": {rec.Artists[0].JoinPhrase},
		"edit-recording.artist_credit.names.0.name":        {rec.Artists[0].NameAsCredited},
		"edit-recording.artist_credit.names.1.artist.name": {rec.Artists[1].Name},
		"edit-recording.artist_credit.names.1.name":        {rec.Artists[1].NameAsCredited},
		"edit-recording.comment":                           {rec.Disambiguation},
		"edit-recording.edit_note":                         {rec.EditNote},
		"edit-recording.isrcs.0":                           {rec.ISRCs[0]},
		"edit-recording.isrcs.1":                           {rec.ISRCs[1]},
		"edit-recording.length":                            {strconv.Itoa(int(rec.Length / time.Millisecond))},
		"edit-recording.name":                              {rec.Name},
		"edit-recording.url.0.text":                        {rec.URLs[0].URL},
		"edit-recording.url.0.link_type_id":                {strconv.Itoa(int(rec.URLs[0].LinkType))},
		"edit-recording.url.1.text":                        {rec.URLs[1].URL},
		"edit-recording.url.1.link_type_id":                {strconv.Itoa(int(rec.URLs[1].LinkType))},
		"rels.0.target":                                    {rel0.Target},
		"rels.0.type":                                      {strconv.Itoa(int(rel0.Type))},
		"rels.0.begin_date":                                {beginDate},
		"rels.0.end_date":                                  {endDate},
		"rels.0.ended":                                     {"1"},
		"rels.0.backward":                                  {"1"},
		"rels.1.type":                                      {rel1.TypeUUID},
		"rels.1.attributes.0.type":                         {strconv.Itoa(int(rel1.Attributes[0].Type))},
		"rels.1.attributes.0.credited_as":                  {rel1.Attributes[0].CreditedAs},
		"rels.1.attributes.1.type":                         {rel1.Attributes[1].TypeUUID},
		"rels.1.attributes.1.text_value":                   {rel1.Attributes[1].TextValue},
	}
	if diff := cmp.Diff(want, rec.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}
}
