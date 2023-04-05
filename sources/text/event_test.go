// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"testing"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/seed"
	"github.com/google/go-cmp/cmp"
)

func TestRead_Event_All(t *testing.T) {
	const (
		beginYear   = 2014
		beginMonth  = 4
		beginDay    = 5
		disambig    = "this one"
		editNote    = "here's my edit"
		endYear     = 2014
		endMonth    = 4
		endDay      = 6
		eventType   = seed.EventType_Concert
		mbid        = "0096a0bf-804e-4e47-bf2a-e0878dbb3eb7"
		name        = "The Concert"
		relTarget   = "dd141ce3-5e5a-48f2-a774-66ba427fca99"
		relType     = seed.LinkType_HeldIn_Area_Event
		relAttrText = "4"
		relAttrType = seed.LinkAttributeType_Number
		setlist     = "First Song\nSecondSong"
		time        = "16:15"
		url         = "https://www.example.org/foo"
		urlType     = seed.LinkType_OfficialHomepage_Event_URL
	)

	var input bytes.Buffer
	if err := csv.NewWriter(&input).WriteAll([][]string{{
		fmt.Sprintf("%04d-%02d-%02d", beginYear, beginMonth, beginDay),
		"true",
		disambig,
		editNote,
		fmt.Sprintf("%04d-%02d-%02d", endYear, endMonth, endDay),
		mbid,
		name,
		relTarget,
		strconv.Itoa(int(relType)),
		relAttrText,
		strconv.Itoa(int(relAttrType)),
		setlist,
		time,
		strconv.Itoa(int(eventType)),
		url,
		strconv.Itoa(int(urlType)),
	}}); err != nil {
		t.Fatal("Failed writing input:", err)
	}
	got, err := Read(context.Background(), &input, CSV, seed.EventEntity, []string{
		"begin_date",
		"cancelled",
		"disambiguation",
		"edit_note",
		"end_date",
		"mbid",
		"name",
		"rel0_target",
		"rel0_type",
		"rel0_attr0_text",
		"rel0_attr0_type",
		"setlist",
		"time",
		"type",
		"url0_url",
		"url0_type",
	}, nil, mbdb.NewDB(mbdb.DisallowQueries))
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Event{
			BeginDate:      seed.MakeDate(beginYear, beginMonth, beginDay),
			Cancelled:      true,
			Disambiguation: disambig,
			EditNote:       editNote,
			EndDate:        seed.MakeDate(endYear, endMonth, endDay),
			MBID:           mbid,
			Name:           name,
			Relationships: []seed.Relationship{{
				Target: relTarget,
				Type:   relType,
				Attributes: []seed.RelationshipAttribute{{
					TextValue: relAttrText,
					Type:      relAttrType,
				}},
			}},
			Setlist: setlist,
			Time:    time,
			Type:    eventType,
			URLs:    []seed.URL{{URL: url, LinkType: urlType}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}
