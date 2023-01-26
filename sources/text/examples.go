// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"bytes"
	"encoding/csv"
	"strings"

	"github.com/derat/yambs/seed"
)

// SetExample returns an example value for the web interface's "set" textarea.
func SetExample(typ seed.Entity) string {
	const editNote = "edit_note=https://www.example.org/"
	switch typ {
	case seed.LabelEntity:
		return "type=7\n" + editNote
	case seed.RecordingEntity:
		return "artist=7e84f845-ac16-41fe-9ff8-df12eb32af55\n" + editNote
	case seed.ReleaseEntity:
		return "language=Eng\nscript=Latn\n" + editNote
	case seed.WorkEntity:
		return "languages=120,1739\n" + editNote
	}
	return ""
}

// FieldsExample returns an example value for the web interface's "fields" input.
func FieldsExample(typ seed.Entity) string {
	switch typ {
	case seed.LabelEntity:
		return "name,begin_date"
	case seed.RecordingEntity:
		return "name,length"
	case seed.ReleaseEntity:
		return "artist0_name,title,status"
	case seed.WorkEntity:
		return "name,type"
	}
	return ""
}

// InputExample returns an example value for the web interface's "input" textarea.
func InputExample(typ seed.Entity, format Format) string {
	if format == KeyVal {
		switch typ {
		case seed.LabelEntity:
			return strings.TrimLeft(`
mbid=02442aba-cf00-445c-877e-f0eaa504d8c2
rel0_target=43bcfb95-f26c-4f8d-84f8-7b2ac5b8ab72
rel0_type=362
rel1_target=a9d8b538-c20a-4025-aea1-5530d616a20a
rel1_type=362
edit_note=https://www.example.org/`, "\n")
		case seed.RecordingEntity:
			return strings.TrimLeft(`
name=Recording Name
artist0_name=Artist Name
length=3:45.04
edit_note=https://www.example.org/`, "\n")
		case seed.ReleaseEntity:
			return strings.TrimLeft(`
title=Album Title
artist0_name=Artist Name
types=Album,Soundtrack
status=Official
packaging=Jewel Case
language=eng
script=Latn
event0_date=2021-05-15
event0_country=XW
medium0_format=CD
medium0_track0_title=First Track
medium0_track0_length=3:45.04
medium0_track1_title=Second Track
medium1_format=CD
medium1_track0_title=First Track on Second Disc
url0_url=https://www.example.org/
url0_type=75
edit_note=https://www.example.org/`, "\n")
		case seed.WorkEntity:
			return strings.TrimLeft(`
name=A Musical
type=29
iswcs=T-123.456.789-0,T-987.654.321-0
edit_note=https://www.example.org/`, "\n")
		}
		return ""
	}

	var rows [][]string
	switch typ {
	case seed.LabelEntity:
		rows = [][]string{
			{"Some Label", "1985-02-13"},
			{"Another Label", "2016"},
		}
	case seed.RecordingEntity:
		rows = [][]string{
			{"Recording Name", "4:35.16"},
			{"Another One", "134500"},
		}
	case seed.ReleaseEntity:
		rows = [][]string{
			{"Artist Name", "Album Title", "Official,Soundtrack"},
			{"Another Artist", "Another Album", "Bootleg"},
		}
	case seed.WorkEntity:
		rows = [][]string{
			{"A Musical", "29"},
			{"An Opera", "10"},
		}
	}

	switch format {
	case CSV:
		var b bytes.Buffer
		csv.NewWriter(&b).WriteAll(rows)
		return b.String()
	case TSV:
		lines := make([]string, len(rows))
		for i := range rows {
			lines[i] = strings.Join(rows[i], "\t")
		}
		return strings.Join(lines, "\n")
	}
	return ""
}
