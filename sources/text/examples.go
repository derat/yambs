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
	case seed.RecordingEntity:
		return "artist=7e84f845-ac16-41fe-9ff8-df12eb32af55\n" + editNote
	case seed.ReleaseEntity:
		return "language=Eng\nscript=Latn\n" + editNote
	}
	return ""
}

// FieldsExample returns an example value for the web interface's "fields" input.
func FieldsExample(typ seed.Entity) string {
	switch typ {
	case seed.RecordingEntity:
		return "name,length"
	case seed.ReleaseEntity:
		return "artist0_name,title,status"
	}
	return ""
}

// InputExample returns an example value for the web interface's "input" textarea.
func InputExample(typ seed.Entity, format Format) string {
	if format == KeyVal {
		switch typ {
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
		}
		return ""
	}

	var rows [][]string
	switch typ {
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
