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

func TestRelease_Params(t *testing.T) {
	rel := Release{
		Title:        "Not of This Earth",
		ReleaseGroup: "502594f4-0502-3976-a2ca-075a9cb7cb8f",
		Types: []ReleaseGroupType{ // usually not set for existing release group
			ReleaseGroupType_Album,
			ReleaseGroupType_Soundtrack, // this album isn't actually a soundtrack
		},
		Disambiguation: "fake release for testing",
		Annotation:     "here's the annotation",
		Barcode:        "none",
		Language:       "eng",
		Script:         "Latn",
		Status:         ReleaseStatus_Official,
		Packaging:      ReleasePackaging_JewelCase,
		Events: []ReleaseEvent{
			{Year: 1986, Month: 5, Day: 1, Country: "JP"},
			{Year: 1990},
		},
		Labels: []ReleaseLabel{
			{MBID: "2bfb17b6-95da-44f8-9cd2-9cc661083901", CatalogNumber: "VZK 90848"},
			{Name: "Some Label"},
		},
		Artists: []ArtistCredit{
			{MBID: "29762c82-bb92-4acd-b1fb-09cc4da250d2", NameAsCredited: "Joe Satriani", JoinPhrase: " & "},
			{Name: "Some Other Artist"},
		},
		Mediums: []Medium{
			{Format: MediumFormat_CD, Tracks: []Track{
				{
					Title:     "Not of This Earth",
					Recording: "9a7502cc-c596-4aec-8e17-75ed537227e2",
					Length:    4*time.Minute + 3*time.Second,
					Artists: []ArtistCredit{
						{MBID: "29762c82-bb92-4acd-b1fb-deadbeefcafe", NameAsCredited: "Joel Satriano", JoinPhrase: " feat. "},
						{Name: "A Third Artist"},
					},
				},
				{Title: "The Snake"},
			}},
			{Name: "Medium Name", Tracks: []Track{
				{Title: "Some Other Track", Number: "2A"},
			}},
		},
		URLs: []URL{
			{URL: "https://www.example.org/foo", LinkType: LinkType_Crowdfunding_Release_URL},
			{URL: "https://www.example.org/bar", LinkType: LinkType_DownloadForFree_Release_URL},
		},
		EditNote: "here's the edit note",
	}

	want := url.Values{
		"annotation":                        {rel.Annotation},
		"artist_credit.names.0.join_phrase": {rel.Artists[0].JoinPhrase},
		"artist_credit.names.0.mbid":        {rel.Artists[0].MBID},
		"artist_credit.names.0.name":        {rel.Artists[0].NameAsCredited},
		"artist_credit.names.1.artist.name": {rel.Artists[1].Name},
		"barcode":                           {rel.Barcode},
		"comment":                           {rel.Disambiguation},
		"edit_note":                         {rel.EditNote},
		"events.0.country":                  {rel.Events[0].Country},
		"events.0.date.day":                 {strconv.Itoa(rel.Events[0].Day)},
		"events.0.date.month":               {strconv.Itoa(rel.Events[0].Month)},
		"events.0.date.year":                {strconv.Itoa(rel.Events[0].Year)},
		"events.1.date.year":                {strconv.Itoa(rel.Events[1].Year)},
		"labels.0.mbid":                     {rel.Labels[0].MBID},
		"labels.0.catalog_number":           {rel.Labels[0].CatalogNumber},
		"labels.1.name":                     {rel.Labels[1].Name},
		"language":                          {rel.Language},
		"mediums.0.format":                  {string(rel.Mediums[0].Format)},

		"mediums.0.track.0.artist_credit.names.0.join_phrase": {rel.Mediums[0].Tracks[0].Artists[0].JoinPhrase},
		"mediums.0.track.0.artist_credit.names.0.mbid":        {rel.Mediums[0].Tracks[0].Artists[0].MBID},
		"mediums.0.track.0.artist_credit.names.0.name":        {rel.Mediums[0].Tracks[0].Artists[0].NameAsCredited},
		"mediums.0.track.0.artist_credit.names.1.artist.name": {rel.Mediums[0].Tracks[0].Artists[1].Name},

		"mediums.0.track.0.length":    {fmt.Sprintf("%d", rel.Mediums[0].Tracks[0].Length.Milliseconds())},
		"mediums.0.track.0.name":      {rel.Mediums[0].Tracks[0].Title},
		"mediums.0.track.0.recording": {rel.Mediums[0].Tracks[0].Recording},
		"mediums.0.track.1.name":      {rel.Mediums[0].Tracks[1].Title},
		"mediums.1.name":              {rel.Mediums[1].Name},
		"mediums.1.track.0.name":      {rel.Mediums[1].Tracks[0].Title},
		"mediums.1.track.0.number":    {rel.Mediums[1].Tracks[0].Number},
		"name":                        {rel.Title},
		"packaging":                   {string(rel.Packaging)},
		"release_group":               {rel.ReleaseGroup},
		"script":                      {rel.Script},
		"status":                      {string(rel.Status)},
		"type":                        {string(rel.Types[0]), string(rel.Types[1])},
		"urls.0.url":                  {rel.URLs[0].URL},
		"urls.0.link_type":            {strconv.Itoa(int(rel.URLs[0].LinkType))},
		"urls.1.url":                  {rel.URLs[1].URL},
		"urls.1.link_type":            {strconv.Itoa(int(rel.URLs[1].LinkType))},
	}
	if diff := cmp.Diff(want, rel.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}

}