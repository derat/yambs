// Copyright 2022 Daniel Erat.
// All rights reserved.

package mp3

import (
	"os"
	"testing"
	"time"

	"github.com/derat/yambs/seed"
	"github.com/google/go-cmp/cmp"
)

func TestReadFile_ID3v24(t *testing.T) {
	const mbid = "7e84f845-ac16-41fe-9ff8-df12eb32af55"
	got, err := getEdits("testdata/id3v24.mp3", seed.ReleaseEntity, []string{
		"artist0_mbid=" + mbid,
		"event0_country=XW",
	})
	if err != nil {
		t.Fatal("Failed creating edits:", err)
	}

	want := []seed.Edit{&seed.Release{
		Title:     "One Second",
		Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
		Language:  "eng",
		Script:    "Latn",
		Status:    seed.ReleaseStatus_Official,
		Packaging: seed.ReleasePackaging_None,
		Events:    []seed.ReleaseEvent{{Country: "XW", Year: 2004}},
		Artists:   []seed.ArtistCredit{{MBID: mbid, NameAsCredited: "Second Artist"}},
		Mediums: []seed.Medium{{
			Format: seed.MediumFormat_DigitalMedia,
			Tracks: []seed.Track{{
				Title:  "One Second",
				Length: 1071 * time.Millisecond,
			}},
		}},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Bad release data:\n" + diff)
	}
}

func TestReadFile_ID3v1(t *testing.T) {
	const editNote = "here's the edit note"
	got, err := getEdits("testdata/id3v1.mp3", seed.RecordingEntity,
		[]string{"edit_note=" + editNote})
	if err != nil {
		t.Fatal("Failed creating edits:", err)
	}

	want := []seed.Edit{&seed.Recording{
		Name:     "Give It Up For ID3v1",
		Artists:  []seed.ArtistCredit{{NameAsCredited: "The Legacy Formats"}},
		Length:   26 * time.Millisecond,
		EditNote: editNote,
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Bad recording data:\n" + diff)
	}
}

func getEdits(p string, typ seed.Entity, rawSetCmds []string) ([]seed.Edit, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadFile(f, typ, rawSetCmds)
}
