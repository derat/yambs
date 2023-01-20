// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package mp3 generates seeded edits from metadata in MP3 files.
package mp3

import (
	"fmt"
	"os"
	"time"

	"github.com/derat/mpeg"
	"github.com/derat/taglib-go/taglib"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/text"
)

// ReadFile reads the passed-in MP3 file and returns an edit of the requested type
// (i.e. either a standalone recording or a "single" release) and additional
// informational edits for any embedded images.
func ReadFile(f *os.File, typ seed.Entity, rawSetCmds []string) ([]seed.Edit, error) {
	setCmds, err := text.ParseSetCommands(rawSetCmds, typ)
	if err != nil {
		return nil, err
	}

	song, err := readSongInfo(f)
	if err != nil {
		return nil, err
	}

	// Create the recording or release edit.
	edit, err := createSongEdit(song, typ)
	if err != nil {
		return nil, err
	}
	for _, pair := range setCmds {
		if err := text.SetField(edit, pair[0], pair[1]); err != nil {
			return nil, fmt.Errorf("failed setting %q: %v", pair[0]+"="+pair[1], err)
		}
	}
	edits := []seed.Edit{edit}

	// Add an informational edit for each embedded image.
	for _, img := range song.images {
		// TODO: These temp image files never get deleted, which feels gross.
		// I'm not sure when they could be safely deleted unless we clean
		// up files from earlier runs that are e.g. more than a day old.
		if ed, err := seed.NewInfo("Embedded image ("+img.desc+")", "file://"+img.path); err != nil {
			return nil, err
		} else {
			edits = append(edits, ed)
		}
	}

	// If we're creating a release and extracted at least one image, redirect to the
	// Add Cover Art page after the release is created.
	if rel, ok := edits[0].(*seed.Release); ok && len(edits) > 1 {
		rel.RedirectURI = seed.AddCoverArtRedirectURI
	}

	return edits, nil
}

// songInfo contains information read from an MP3 file's ID3 tags.
type songInfo struct {
	artist string
	title  string
	album  string
	length time.Duration
	time   mpeg.Time
	images []imgInfo
}

// readSongInfo returns a songInfo object based on the supplied MP3 file.
func readSongInfo(f *os.File) (*songInfo, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	var song songInfo
	var headerLen, footerLen int64

	if v1, err := mpeg.ReadID3v1Footer(f, fi); err != nil {
		return nil, err
	} else if v1 != nil {
		// I'm ignoring the year since ID3v1 data quality is usually pretty poor.
		// It might be better to just ignore *all* ID3v1 data....
		song.artist = v1.Artist
		song.title = v1.Title
		song.album = v1.Album
		footerLen = mpeg.ID3v1Length
	}

	if v2, err := taglib.Decode(f, fi.Size()); err != nil {
		// Tolerate missing ID3v2 tags if we got an artist and title from ID3v1.
		if len(song.artist) == 0 && len(song.title) == 0 {
			return nil, err
		}
	} else {
		// TODO: Use v2.CustomFrames to report an error if the file already has an MBID?
		song.artist = v2.Artist()
		song.title = v2.Title()
		song.album = v2.Album()

		for _, tt := range []mpeg.TimeType{mpeg.ReleaseTime, mpeg.RecordingTime} {
			if tm, err := mpeg.GetID3v2Time(v2, tt); err != nil {
				return nil, err
			} else if !tm.Empty() {
				song.time = tm
				break
			}
		}

		var err error
		if song.images, err = getImages(v2); err != nil {
			return nil, err
		}
		headerLen = int64(v2.TagSize())
	}

	song.length, _, _, err = mpeg.ComputeAudioDuration(f, fi, headerLen, footerLen)
	if err != nil {
		return nil, err
	}
	return &song, nil
}

// MP3 release date per https://en.wikipedia.org/wiki/MP3.
var mp3RelDate = time.Date(1991, 12, 6, 0, 0, 0, 0, time.UTC)

// createSongEdit creates a seed.Edit of the requested type based on the supplied song.
func createSongEdit(song *songInfo, typ seed.Entity) (seed.Edit, error) {
	switch typ {
	case seed.RecordingEntity:
		return &seed.Recording{
			Name:    song.title,
			Artists: []seed.ArtistCredit{{NameAsCredited: song.artist}},
			Length:  song.length,
		}, nil

	case seed.ReleaseEntity:
		rel := seed.Release{
			Title: song.title,
			Types: []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
			// TODO: Infer these?
			Language:  "eng",
			Script:    "Latn",
			Status:    seed.ReleaseStatus_Official,
			Packaging: seed.ReleasePackaging_None,
			Artists:   []seed.ArtistCredit{{NameAsCredited: song.artist}},
			Mediums: []seed.Medium{{
				Format: seed.MediumFormat_DigitalMedia,
				Tracks: []seed.Track{{
					Title:  song.title,
					Length: song.length,
				}},
			}},
		}
		if tm := song.time; !tm.Empty() && !tm.Time().Before(mp3RelDate) {
			var ev seed.ReleaseEvent
			if year := tm.Year(); year >= 1 {
				ev.Year = year
			}
			if month := tm.Month(); month >= 1 {
				ev.Month = month
			}
			if day := tm.Day(); day >= 1 {
				ev.Day = day
			}
			rel.Events = append(rel.Events, ev)
		}
		return &rel, nil

	default:
		return nil, fmt.Errorf("unsupported type %q", typ)
	}
}
