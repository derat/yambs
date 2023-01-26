// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// labelFields defines fields that can be set in a seed.Label.
var labelFields = map[string]fieldInfo{
	"area_name": {
		"Area name to prefill search",
		func(l *seed.Label, k, v string) error { return setString(&l.AreaName, v) },
	},
	"begin_date": {
		`Date when label began as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(l *seed.Label, k, v string) error { return setDate(&l.BeginDate, v) },
	},
	"disambiguation": {
		"Comment disambiguating this label from others with similar names",
		func(l *seed.Label, k, v string) error { return setString(&l.Disambiguation, v) },
	},
	"edit_note": {
		"Note attached to edit",
		func(l *seed.Label, k, v string) error { return setString(&l.EditNote, v) },
	},
	"end_date": {
		`Date when label ended as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(l *seed.Label, k, v string) error { return setDate(&l.EndDate, v) },
	},
	"ended": {
		`Whether the label has ended ("1" or "true" if true)`,
		func(l *seed.Label, k, v string) error { return setBool(&l.Ended, v) },
	},
	"ipi_codes": {
		"Comma-separated IPI (Interested Party Information) codes",
		func(l *seed.Label, k, v string) error { return setStringSlice(&l.IPICodes, v, ",") },
	},
	"isni_codes": {
		"Comma-separated ISNI (International Standard Name Identifier) codes",
		func(l *seed.Label, k, v string) error { return setStringSlice(&l.ISNICodes, v, ",") },
	},
	"label_code": {
		`Label's 4- or 5-digit label code (without "LC-" prefix)`,
		func(l *seed.Label, k, v string) error { return setString(&l.LabelCode, v) },
	},
	"mbid": {
		"MBID of existing label to edit (if empty, create label)",
		func(l *seed.Label, k, v string) error { return setMBID(&l.MBID, v) },
	},
	"name": {
		"Label's name",
		func(l *seed.Label, k, v string) error { return setString(&l.Name, v) },
	},
	"type": {
		"Integer [label type](" + labelTypeURL + ") describing label's main activity",
		func(l *seed.Label, k, v string) error { return setInt((*int)(&l.Type), v) },
	},
}

func init() {
	// Add common fields.
	addRelationshipFields(labelFields,
		func(fn relFunc) interface{} {
			return func(l *seed.Label, k, v string) error {
				return indexedField(&l.Relationships, k, "rel",
					func(rel *seed.Relationship) error { return fn(rel, k, v) })
			}
		})
	addURLFields(labelFields,
		func(fn urlFunc) interface{} {
			return func(l *seed.Label, k, v string) error {
				return indexedField(&l.URLs, k, "url",
					func(url *seed.URL) error { return fn(url, v) })
			}
		})
}
