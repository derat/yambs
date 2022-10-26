// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// releaseFields defines fields that can be set in a seed.Release.
var releaseFields = map[string]fieldInfo{
	"artist": {
		"MBID of artist receiving primary credit for release",
		func(r *seed.Release, k, v string) error { return setString(&r.Artist, v) },
	},
	"date": {
		`Release date as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(r *seed.Release, k, v string) error { return setDate(&r.Date, v) },
	},
	"title": {
		"Release's name",
		func(r *seed.Release, k, v string) error { return setString(&r.Title, v) },
	},
}
