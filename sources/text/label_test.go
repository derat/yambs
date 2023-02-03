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

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/google/go-cmp/cmp"
)

func TestRead_Label_All(t *testing.T) {
	const (
		areaName    = "New York"
		beginYear   = 2014
		beginMonth  = 4
		beginDay    = 5
		disambig    = "this one"
		editNote    = "here's my edit"
		endYear     = 2018
		endMonth    = 12
		endDay      = 31
		ipi1        = "123456789"
		ipi2        = "987654321"
		isni1       = "1234567899999799"
		isni2       = "000000012146438X"
		labelCode   = "52361"
		labelType   = seed.LabelType_Manufacturer
		mbid        = "0096a0bf-804e-4e47-bf2a-e0878dbb3eb7"
		name        = "The Label"
		relTarget   = "65389277-491a-4055-8e71-0a9be1c9c99c"
		relType     = seed.LinkType_Manufactured_Label_Release
		relAttrText = "4"
		relAttrType = seed.LinkAttributeType_Number
		url         = "https://www.example.org/foo"
		urlType     = seed.LinkType_DownloadForFree_Label_URL
	)

	var input bytes.Buffer
	if err := csv.NewWriter(&input).WriteAll([][]string{{
		areaName,
		fmt.Sprintf("%04d-%02d-%02d", beginYear, beginMonth, beginDay),
		disambig,
		editNote,
		fmt.Sprintf("%04d-%02d-%02d", endYear, endMonth, endDay),
		"true",
		ipi1 + "," + ipi2,
		isni1 + "," + isni2,
		labelCode,
		mbid,
		name,
		relTarget,
		strconv.Itoa(int(relType)),
		relAttrText,
		strconv.Itoa(int(relAttrType)),
		url,
		strconv.Itoa(int(urlType)),
		strconv.Itoa(int(labelType)),
	}}); err != nil {
		t.Fatal("Failed writing input:", err)
	}
	got, err := Read(context.Background(), &input, CSV, seed.LabelEntity, []string{
		"area_name",
		"begin_date",
		"disambiguation",
		"edit_note",
		"end_date",
		"ended",
		"ipi_codes",
		"isni_codes",
		"label_code",
		"mbid",
		"name",
		"rel0_target",
		"rel0_type",
		"rel0_attr0_text",
		"rel0_attr0_type",
		"url0_url",
		"url0_type",
		"type",
	}, nil, db.NewDB(db.DisallowQueries))
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Label{
			AreaName:       areaName,
			BeginDate:      seed.MakeDate(beginYear, beginMonth, beginDay),
			Disambiguation: disambig,
			EditNote:       editNote,
			EndDate:        seed.MakeDate(endYear, endMonth, endDay),
			Ended:          true,
			IPICodes:       []string{ipi1, ipi2},
			ISNICodes:      []string{isni1, isni2},
			LabelCode:      labelCode,
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
			Type: labelType,
			URLs: []seed.URL{{URL: url, LinkType: urlType}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}
