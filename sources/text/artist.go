// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// artistFields defines fields that can be set in a seed.Artist.
var artistFields = map[string]fieldInfo{
	"area_name": {
		"Area name to prefill search",
		func(a *seed.Artist, k, v string) error { return setString(&a.AreaName, v) },
	},
	"begin_area_name": {
		"Begin area name to prefill search",
		func(a *seed.Artist, k, v string) error { return setString(&a.BeginAreaName, v) },
	},
	"begin_date": {
		`Date when artist began as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(a *seed.Artist, k, v string) error { return setDate(&a.BeginDate, v) },
	},
	"disambiguation": {
		"Comment disambiguating this artist from others with similar names",
		func(a *seed.Artist, k, v string) error { return setString(&a.Disambiguation, v) },
	},
	"edit_note": {
		"Note attached to edit",
		func(a *seed.Artist, k, v string) error { return setString(&a.EditNote, v) },
	},
	"end_area_name": {
		"End area name to prefill search",
		func(a *seed.Artist, k, v string) error { return setString(&a.EndAreaName, v) },
	},
	"end_date": {
		`Date when artist ended as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(a *seed.Artist, k, v string) error { return setDate(&a.EndDate, v) },
	},
	"ended": {
		`Whether the artist has ended ("1" or "true" if true)`,
		func(a *seed.Artist, k, v string) error { return setBool(&a.Ended, v) },
	},
	"gender": {
		"Integer [gender ID](" + genderURL + ") describing artist's gender",
		func(a *seed.Artist, k, v string) error { return setInt((*int)(&a.Gender), v) },
	},
	"ipi_codes": {
		"Comma-separated IPI (Interested Party Information) codes",
		func(a *seed.Artist, k, v string) error { return setStringSlice(&a.IPICodes, v, ",") },
	},
	"isni_codes": {
		"Comma-separated ISNI (International Standard Name Identifier) codes",
		func(a *seed.Artist, k, v string) error { return setStringSlice(&a.ISNICodes, v, ",") },
	},
	"mbid": {
		"MBID of existing artist to edit (if empty, create artist)",
		func(a *seed.Artist, k, v string) error { return setMBID(&a.MBID, v) },
	},
	"name": {
		"Artist's name",
		func(a *seed.Artist, k, v string) error { return setString(&a.Name, v) },
	},
	"sort_name": {
		"Artist's sort name",
		func(a *seed.Artist, k, v string) error { return setString(&a.SortName, v) },
	},
	"type": {
		"Integer [artist type](" + artistTypeURL + ")",
		func(a *seed.Artist, k, v string) error { return setInt((*int)(&a.Type), v) },
	},
}

func init() {
	// Add common fields.
	addRelationshipFields(artistFields,
		func(fn relFunc) interface{} {
			return func(a *seed.Artist, k, v string) error {
				return indexedField(&a.Relationships, k, "rel",
					func(rel *seed.Relationship) error { return fn(rel, k, v) })
			}
		})
	addURLFields(artistFields,
		func(fn urlFunc) interface{} {
			return func(a *seed.Artist, k, v string) error {
				return indexedField(&a.URLs, k, "url",
					func(url *seed.URL) error { return fn(url, v) })
			}
		})
}
