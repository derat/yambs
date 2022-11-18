// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/google/go-cmp/cmp"
)

func TestRead_Recording_Multiple(t *testing.T) {
	const (
		uuid  = "b92d909c-243d-4146-bfd5-2703c9dd1c99"
		note  = "From https://www.example.org"
		input = `
First Song	4:56
Second Song	2:12
Third Song	0:45
`
	)
	got, err := Read(context.Background(), strings.NewReader(strings.TrimLeft(input, "\n")),
		TSV, seed.RecordingType, []string{"name", "length"}, []string{
			"artist=" + uuid,
			"edit_note=" + note,
		}, db.NewDB(db.DisallowQueries))
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Recording{
			Name:     "First Song",
			Artist:   uuid,
			Length:   4*time.Minute + 56*time.Second,
			EditNote: note,
		},
		&seed.Recording{
			Name:     "Second Song",
			Artist:   uuid,
			Length:   2*time.Minute + 12*time.Second,
			EditNote: note,
		},
		&seed.Recording{
			Name:     "Third Song",
			Artist:   uuid,
			Length:   45 * time.Second,
			EditNote: note,
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}

func TestRead_Recording_All(t *testing.T) {
	const (
		artistMBID = "b92d909c-243d-4146-bfd5-2703c9dd1c99"
		artistID   = 1234
		artistCred = "The Artist"
		artistJoin = " & "
		artistName = "Other Artist"
		disambig   = "Different from the other one"
		editNote   = "From https://www.example.org"
		isrc1      = "UKAAA0500001"
		isrc2      = "USBBB0400002"
		recMBID    = "5e1a028f-461d-4ec8-aa10-97c4cb7262dc"
		recName    = "Recording Name"
		url        = "https://www.example.org/foo"
		url2       = "https://www.example.org/bar"
		linkType   = seed.LinkType_DownloadForFree_Recording_URL
	)

	db := db.NewDB(db.DisallowQueries)
	db.SetDatabaseIDForTest(artistMBID, artistID)

	var input bytes.Buffer
	if err := csv.NewWriter(&input).WriteAll([][]string{{
		artistMBID,
		artistCred,
		artistJoin,
		artistName,
		disambig,
		editNote,
		isrc1 + "," + isrc2,
		"3:45",
		recMBID,
		recName,
		"true",
		url,
		strconv.Itoa(int(linkType)),
		url2,
	}}); err != nil {
		t.Fatal("Failed writing input:", err)
	}
	got, err := Read(context.Background(), &input, CSV, seed.RecordingType, []string{
		"artist0_mbid",
		"artist0_credited",
		"artist0_join",
		"artist1_name",
		"disambig", // prefix
		"edit_note",
		"isrcs",
		"length",
		"mbid",
		"name",
		"video",
		"url0_url",
		"url0_type",
		"url1_url",
	}, nil, db)
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Recording{
			Artists: []seed.ArtistCredit{
				{ID: artistID, NameAsCredited: artistCred, JoinPhrase: artistJoin},
				{Name: artistName},
			},
			Disambiguation: disambig,
			EditNote:       editNote,
			ISRCs:          []string{isrc1, isrc2},
			Length:         3*time.Minute + 45*time.Second,
			MBID:           recMBID,
			Name:           recName,
			Video:          true,
			URLs:           []seed.URL{{URL: url, LinkType: linkType}, {URL: url2}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}

func TestRead_Recording_BadIndex(t *testing.T) {
	// Read should reject an "artist1" field if "artist0" wasn't previously supplied.
	ctx := context.Background()
	db := db.NewDB(db.DisallowQueries)
	if _, err := Read(ctx,
		strings.NewReader("name=Name\nartist1_credited=Artist\n"),
		KeyVal, seed.RecordingType, nil, nil, db); err == nil {
		t.Fatal("Read unexpectedly accepted input with large index")
	}

	// Check that things work if indexed fields are given in-order.
	if _, err := Read(ctx,
		strings.NewReader("name=Name\nartist0_credited=Artist\nartist1_credited=Artist\n"),
		KeyVal, seed.RecordingType, nil, nil, db); err != nil {
		t.Fatal("Read failed:", err)
	}
}

func TestRead_Recording_MaxEdits(t *testing.T) {
	// Read should accept input matching the maximum number of edits.
	ctx := context.Background()
	db := db.NewDB(db.DisallowQueries)
	opt := MaxEdits(2)
	if _, err := Read(ctx, strings.NewReader("Name 1\nName 2\n"),
		TSV, seed.RecordingType, []string{"name"}, nil, db, opt); err != nil {
		t.Fatal("Read failed:", err)
	}

	// It should return an error if too many edits are supplied.
	if _, err := Read(ctx, strings.NewReader("Name 1\nName 2\nName 3\n"),
		TSV, seed.RecordingType, []string{"name"}, nil, db, opt); err == nil {
		t.Fatal("Read unexpectedly accepted input with too many edits")
	}
}

func TestRead_Recording_MaxFields(t *testing.T) {
	// Read should accept input matching the maximum number of fields.
	ctx := context.Background()
	db := db.NewDB(db.DisallowQueries)
	opt := MaxFields(2)
	if _, err := Read(ctx, strings.NewReader("Name\t3:45"), TSV, seed.RecordingType,
		[]string{"name", "length"}, nil, db, opt); err != nil {
		t.Fatal("Read failed:", err)
	}

	// It should return an error if too many fields are supplied.
	if _, err := Read(ctx, strings.NewReader("Name\tArtist\t3:45"), TSV, seed.RecordingType,
		[]string{"name", "artist0_name", "length"}, nil, db, opt); err == nil {
		t.Fatal("Read unexpectedly accepted input with too many fields")
	}

	// Set commands should count toward the limit too.
	if _, err := Read(ctx, strings.NewReader("Name\t3:45"), TSV, seed.RecordingType,
		[]string{"name", "length"}, []string{"artist0_name=Artist"}, db, opt); err == nil {
		t.Fatal("Read unexpectedly accepted input with too many fields (including set commands)")
	}

	// Slash-separated fields should be counted as well.
	if _, err := Read(ctx, strings.NewReader("Name\t3:45"), TSV, seed.RecordingType,
		[]string{"name/artist0_name", "length"}, nil, db, opt); err == nil {
		t.Fatal("Read unexpectedly accepted input with too many fields (slash-separated)")
	}
}

func TestRead_Recording_MultipleFields(t *testing.T) {
	// Check that multiple slash-separated fields can be specified.
	const (
		url1 = "https://example.org/foo"
		url2 = "https://example.org/bar"
	)
	got, err := Read(context.Background(),
		strings.NewReader("Name 1\t"+url1+"\nName 2\t"+url2+"\n"),
		TSV, seed.RecordingType, []string{"name", "url0_url/edit_note"}, nil,
		db.NewDB(db.DisallowQueries))
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Recording{Name: "Name 1", URLs: []seed.URL{{URL: url1}}, EditNote: url1},
		&seed.Recording{Name: "Name 2", URLs: []seed.URL{{URL: url2}}, EditNote: url2},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}

func TestRead_Recording_SkipField(t *testing.T) {
	// Check that an empty field name can be passed to skip the corresponding column.
	got, err := Read(context.Background(),
		strings.NewReader("Name 1\tfoo\t3:56\nName 2\tbar\t0:45\n"),
		TSV, seed.RecordingType, []string{"name", "", "length"}, nil,
		db.NewDB(db.DisallowQueries))
	if err != nil {
		t.Fatal("Read failed:", err)
	}

	want := []seed.Edit{
		&seed.Recording{Name: "Name 1", Length: 3*time.Minute + 56*time.Second},
		&seed.Recording{Name: "Name 2", Length: 45 * time.Second},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Read returned wrong edits:\n" + diff)
	}
}
