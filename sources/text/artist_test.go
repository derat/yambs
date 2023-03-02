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

func TestRead_Artist_All(t *testing.T) {
	const (
		areaName        = "New York"
		artistType      = seed.ArtistType_Person
		beginAreaName   = "Kentucky"
		beginYear       = 1956
		beginMonth      = 4
		beginDay        = 5
		disambig        = "this one"
		editNote        = "here's my edit"
		endAreaName     = "France"
		endYear         = 2018
		endMonth        = 12
		endDay          = 31
		gender          = seed.Gender_Female
		ipi1            = "123456789"
		ipi2            = "987654321"
		isni1           = "1234567899999799"
		isni2           = "000000012146438X"
		mbid            = "0096a0bf-804e-4e47-bf2a-e0878dbb3eb7"
		name            = "Ms. Musician"
		relSourceCredit = "Source Credit"
		relTarget       = "65389277-491a-4055-8e71-0a9be1c9c99c"
		relTargetCredit = "Target Credit"
		relType         = seed.LinkType_Arranger_Artist_Release
		relAttrText     = "4"
		relAttrType     = seed.LinkAttributeType_Number
		sortName        = "Musician, Ms."
		url             = "https://example.bandcamp.com/"
		urlType         = seed.LinkType_Bandcamp_Artist_URL
	)

	var input bytes.Buffer
	if err := csv.NewWriter(&input).WriteAll([][]string{{
		areaName,
		beginAreaName,
		fmt.Sprintf("%04d-%02d-%02d", beginYear, beginMonth, beginDay),
		disambig,
		editNote,
		endAreaName,
		fmt.Sprintf("%04d-%02d-%02d", endYear, endMonth, endDay),
		"true",
		strconv.Itoa(int(gender)),
		ipi1 + "," + ipi2,
		isni1 + "," + isni2,
		mbid,
		name,
		sortName,
		relTarget,
		strconv.Itoa(int(relType)),
		relSourceCredit,
		relTargetCredit,
		relAttrText,
		strconv.Itoa(int(relAttrType)),
		url,
		strconv.Itoa(int(urlType)),
		strconv.Itoa(int(artistType)),
	}}); err != nil {
		t.Fatal("Failed writing input:", err)
	}
	got, err := Read(context.Background(), &input, CSV, seed.ArtistEntity, []string{
		"area_name",
		"begin_area_name",
		"begin_date",
		"disambiguation",
		"edit_note",
		"end_area_name",
		"end_date",
		"ended",
		"gender",
		"ipi_codes",
		"isni_codes",
		"mbid",
		"name",
		"sort_name",
		"rel0_target",
		"rel0_type",
		"rel0_source_credit",
		"rel0_target_credit",
		"rel0_attr0_text",
		"rel0_attr0_type",
		"url0_url",
		"url0_type",
		"type",
	}, nil, mbdb.NewDB(mbdb.DisallowQueries))
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Artist{
			AreaName:       areaName,
			BeginAreaName:  beginAreaName,
			BeginDate:      seed.MakeDate(beginYear, beginMonth, beginDay),
			Disambiguation: disambig,
			EditNote:       editNote,
			EndAreaName:    endAreaName,
			EndDate:        seed.MakeDate(endYear, endMonth, endDay),
			Ended:          true,
			Gender:         gender,
			IPICodes:       []string{ipi1, ipi2},
			ISNICodes:      []string{isni1, isni2},
			MBID:           mbid,
			Name:           name,
			Relationships: []seed.Relationship{{
				Target:       relTarget,
				Type:         relType,
				SourceCredit: relSourceCredit,
				TargetCredit: relTargetCredit,
				Attributes: []seed.RelationshipAttribute{{
					TextValue: relAttrText,
					Type:      relAttrType,
				}},
			}},
			SortName: sortName,
			Type:     artistType,
			URLs:     []seed.URL{{URL: url, LinkType: urlType}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}
