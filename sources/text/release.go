// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

type releaseFunc func(r *seed.Release, k, v string) error

// releaseFields maps from user-supplied field names to functions that set the appropriate
// field in a seed.Release.
var releaseFields = map[string]releaseFunc{
	"artist": func(r *seed.Release, k, v string) error { return setString(&r.Artist, v) },
	"date":   func(r *seed.Release, k, v string) error { return setDate(&r.Date, v) },
	"title":  func(r *seed.Release, k, v string) error { return setString(&r.Title, v) },
}
