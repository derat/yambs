// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"testing"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/seed"
	"github.com/google/go-cmp/cmp"
)

func TestRead_Work_All(t *testing.T) {
	const (
		attrType    = seed.WorkAttributeType_ASCAP_ID
		attrValue   = "123456789"
		disambig    = "not the same"
		editNote    = "here's the justification"
		iswc1       = "T-345246800-1"
		iswc2       = "T-123456789-0"
		lang1       = seed.Language_Hindi
		lang2       = seed.Language_Hebrew
		mbid        = "0096a0bf-804e-4e47-bf2a-e0878dbb3eb7"
		name        = "Work Title"
		relTarget   = "65389277-491a-4055-8e71-0a9be1c9c99c"
		relType     = seed.LinkType_BasedOn_Work_Work
		relAttrText = "4"
		relAttrType = seed.LinkAttributeType_Number
		url         = "https://www.example.org/foo"
		urlType     = seed.LinkType_Lyrics_URL_Work
		workType    = seed.WorkType_Mass
	)

	var input bytes.Buffer
	if err := csv.NewWriter(&input).WriteAll([][]string{{
		strconv.Itoa(int(attrType)),
		attrValue,
		disambig,
		editNote,
		iswc1 + "," + iswc2,
		strconv.Itoa(int(lang1)) + "," + strconv.Itoa(int(lang2)),
		mbid,
		name,
		relTarget,
		strconv.Itoa(int(relType)),
		relAttrText,
		strconv.Itoa(int(relAttrType)),
		url,
		strconv.Itoa(int(urlType)),
		strconv.Itoa(int(workType)),
	}}); err != nil {
		t.Fatal("Failed writing input:", err)
	}
	got, err := Read(context.Background(), &input, CSV, seed.WorkEntity, []string{
		"attr0_type",
		"attr0_value",
		"disambiguation",
		"edit_note",
		"iswcs",
		"languages",
		"mbid",
		"name",
		"rel0_target",
		"rel0_type",
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
		&seed.Work{
			Attributes: []seed.WorkAttribute{{
				Type:  attrType,
				Value: attrValue,
			}},
			Disambiguation: disambig,
			EditNote:       editNote,
			ISWCs:          []string{iswc1, iswc2},
			Languages:      []seed.Language{lang1, lang2},
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
			URLs: []seed.URL{{URL: url, LinkType: urlType}},
			Type: workType,
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}
