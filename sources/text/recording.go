// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
)

const maxArtistCredits = 100

type recordingFunc func(r *seed.Recording, k, v string) error

// recordingFields maps from user-supplied field names to functions that set the appropriate
// field in a seed.Recording.
var recordingFields = map[string]recordingFunc{
	"artist":         func(r *seed.Recording, k, v string) error { return setString(&r.Artist, v) },
	"disambiguation": func(r *seed.Recording, k, v string) error { return setString(&r.Disambiguation, v) },
	"edit_note":      func(r *seed.Recording, k, v string) error { return setString(&r.EditNote, v) },
	"isrcs":          func(r *seed.Recording, k, v string) error { return setStringSlice(&r.ISRCs, v, ",") },
	"length":         func(r *seed.Recording, k, v string) error { return setDuration(&r.Length, v) },
	"mbid":           func(r *seed.Recording, k, v string) error { return setString(&r.MBID, v) },
	"title":          func(r *seed.Recording, k, v string) error { return setString(&r.Title, v) },
	"video":          func(r *seed.Recording, k, v string) error { return setBool(&r.Video, v) },

	"artist*_mbid": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		// The /recording/create form only seems to accept database IDs, not MBIDs.
		// TODO: Pass a context in, maybe.
		ac.ID, err = db.GetArtistID(context.Background(), v)
		return err
	},
	"artist*_name": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		return setString(&ac.Name, v)
	},
	"artist*_credited": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		return setString(&ac.NameAsCredited, v)
	},
	"artist*_join_phrase": func(r *seed.Recording, k, v string) error {
		ac, err := getArtistCredit(r, k)
		if err != nil {
			return err
		}
		return setString(&ac.JoinPhrase, v)
	},
}

var artistFieldRegexp = regexp.MustCompile(`^artist(\d*)_.*`)

// getArtistCredit extracts a zero-based index from field (e.g. "artist3_name") and returns
// a pointer to the corresponding item from rec.ArtistCredits, creating empty items if needed.
// The index 0 is inferred for e.g. "artist_name".
func getArtistCredit(rec *seed.Recording, field string) (*seed.ArtistCredit, error) {
	matches := artistFieldRegexp.FindStringSubmatch(field)
	if matches == nil {
		return nil, &fieldNameError{`field doesn't start with "artist_" or "artist<num>_"`}
	}
	var idx int
	if matches[1] != "" {
		var err error
		if idx, err = strconv.Atoi(matches[1]); err != nil {
			return nil, err
		} else if idx >= maxArtistCredits {
			// Keep the user from shooting themselves in the foot.
			return nil, &fieldNameError{fmt.Sprintf("invalid artist index %d", idx)}
		}
	}
	if idx >= len(rec.ArtistCredits) {
		old := rec.ArtistCredits
		rec.ArtistCredits = make([]seed.ArtistCredit, idx+1)
		copy(rec.ArtistCredits, old)
	}
	return &rec.ArtistCredits[idx], nil
}
