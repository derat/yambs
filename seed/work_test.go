// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWork_Params(t *testing.T) {
	work := Work{
		Name:           "Time",
		Disambiguation: "for testing",
		Type:           WorkType_Song,
		Languages:      []Language{Language_English},
		ISWCs:          []string{"T-010.455.103-2"},
		Attributes: []WorkAttribute{{
			Type:  WorkAttributeType_ASCAP_ID,
			Value: "500220308",
		}},
		Relationships: []Relationship{{
			Target: "0f50beab-d77d-4f0f-ac26-0b87d3e9b11b",
			Type:   LinkType_Lyricist_Artist_Work,
		}},
		URLs: []URL{{
			URL:      "https://www.wikidata.org/wiki/Q641913",
			LinkType: LinkType_Wikidata_URL_Work,
		}},
		EditNote: "here's the edit note",
	}

	attr := work.Attributes[0]
	rel := work.Relationships[0]
	want := url.Values{
		"edit-work.name":                 {work.Name},
		"edit-work.comment":              {work.Disambiguation},
		"edit-work.type_id":              {strconv.Itoa(int(work.Type))},
		"edit-work.languages.0":          {strconv.Itoa(int(work.Languages[0]))},
		"edit-work.iswcs.0":              {work.ISWCs[0]},
		"edit-work.attributes.0.type_id": {strconv.Itoa(int(attr.Type))},
		"edit-work.attributes.0.value":   {attr.Value},
		"rels.0.target":                  {rel.Target},
		"rels.0.type":                    {strconv.Itoa(int(rel.Type))},
		"edit-work.url.0.text":           {work.URLs[0].URL},
		"edit-work.url.0.link_type_id":   {strconv.Itoa(int(work.URLs[0].LinkType))},
		"edit-work.edit_note":            {work.EditNote},
	}
	if diff := cmp.Diff(want, work.Params()); diff != "" {
		t.Error("Incorrect query params:\n" + diff)
	}
}
