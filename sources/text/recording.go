// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// recordingFields defines fields that can be set in a seed.Recording.
var recordingFields = map[string]fieldInfo{
	"artist": {
		"MBID of artist receiving primary credit for recording",
		func(r *seed.Recording, k, v string) error { return setMBID(&r.Artist, v) },
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
		func(r *seed.Recording, k, v string) error { return setMBID(&r.MBID, v) },
	},
	"name": {
		"Recording's name",
		func(r *seed.Recording, k, v string) error { return setString(&r.Name, v) },
	},
	"video": {
		`Whether this is a video recording ("1" or "true" if true)`,
		func(r *seed.Recording, k, v string) error { return setBool(&r.Video, v) },
	},
}

func init() {
	// Add common fields.
	addArtistCreditFields(recordingFields, "",
		func(fn artistFunc) interface{} {
			return func(r *seed.Recording, k, v string) error {
				return indexedField(&r.Artists, k, "artist",
					func(ac *seed.ArtistCredit) error { return fn(ac, v) })
			}
		})
	addRelationshipFields(recordingFields,
		func(fn relFunc) interface{} {
			return func(r *seed.Recording, k, v string) error {
				return indexedField(&r.Relationships, k, "rel",
					func(rel *seed.Relationship) error { return fn(rel, k, v) })
			}
		})
	addURLFields(recordingFields,
		func(fn urlFunc) interface{} {
			return func(r *seed.Recording, k, v string) error {
				return indexedField(&r.URLs, k, "url",
					func(url *seed.URL) error { return fn(url, v) })
			}
		})
}
