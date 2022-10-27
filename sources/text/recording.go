// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"context"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
)

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
		"MBID of existing recording to edit (if empty, create recording)",
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
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
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
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.Name, v)
		},
	},
	"artist*_credited": {
		"As-credited name of 0-indexed artist",
		func(r *seed.Recording, k, v string) error {
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.NameAsCredited, v)
		},
	},
	"artist*_join_phrase": {
		`Join phrase used to separate 0-indexed artist and next artist (e.g. " & ")`,
		func(r *seed.Recording, k, v string) error {
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.JoinPhrase, v)
		},
	},
}
