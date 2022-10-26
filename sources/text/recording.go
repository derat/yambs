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

// recordingFields defines fields that can be set in a seed.Recording.
var recordingFields = map[string]fieldInfo{
	"artist": {
		"MBID of artist receiving primary credit for recording",
		func(r *seed.Recording, k, v string) error { return setString(&r.Artist, v) },
	},
	"disambiguation": {
		"Comment disambiguating this recording from others with similar names",
		func(r *seed.Recording, k, v string) error { return setString(&r.Disambiguation, v) },
	},
	"edit_note": {
		"Note attached to edit",
		func(r *seed.Recording, k, v string) error { return setString(&r.EditNote, v) },
	},
	"isrcs": {
		"Comma-separated ISRCs identifying recording",
		func(r *seed.Recording, k, v string) error { return setStringSlice(&r.ISRCs, v, ",") },
	},
	"length": {
		`Recording's duration as e.g. "3:45.01" or total milliseconds`,
		func(r *seed.Recording, k, v string) error { return setDuration(&r.Length, v) },
	},
	"mbid": {
		"MBID of existing recording to edit (if empty, create a recording)",
		func(r *seed.Recording, k, v string) error { return setString(&r.MBID, v) },
	},
	"title": {
		"Recording's name",
		func(r *seed.Recording, k, v string) error { return setString(&r.Title, v) },
	},
	"video": {
		`Whether this is a video recording ("1" or "true" if true)`,
		func(r *seed.Recording, k, v string) error { return setBool(&r.Video, v) },
	},
	"artist*_mbid": {
		"MBID of 0-indexed artist",
		func(r *seed.Recording, k, v string) error {
			ac, err := getArtistCredit(r, k)
			if err != nil {
				return err
			}
			// The /recording/create form only seems to accept database IDs, not MBIDs.
			// TODO: Pass a context in, maybe.
			ac.ID, err = db.GetArtistID(context.Background(), v)
			return err
		},
	},
	"artist*_name": {
		"MusicBrainz name of 0-indexed artist",
		func(r *seed.Recording, k, v string) error {
			ac, err := getArtistCredit(r, k)
			if err != nil {
				return err
			}
			return setString(&ac.Name, v)
		},
	},
	"artist*_credited": {
		"As-credited name of 0-indexed artist",
		func(r *seed.Recording, k, v string) error {
			ac, err := getArtistCredit(r, k)
			if err != nil {
				return err
			}
			return setString(&ac.NameAsCredited, v)
		},
	},
	"artist*_join_phrase": {
		`Join phrase used to separate 0-indexed artist and next artist (e.g. " & ")`,
		func(r *seed.Recording, k, v string) error {
			ac, err := getArtistCredit(r, k)
			if err != nil {
				return err
			}
			return setString(&ac.JoinPhrase, v)
		},
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
