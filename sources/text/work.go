// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"strconv"
	"strings"

	"github.com/derat/yambs/seed"
)

// workFields defines fields that can be set in a seed.Work.
var workFields = map[string]fieldInfo{
	"attr*_type": {
		"Integer [work attribute type](" + workAttrTypeURL + ") describing attribute",
		func(w *seed.Work, k, v string) error {
			return workAttribute(w, k, func(a *seed.WorkAttribute) error { return setInt((*int)(&a.Type), v) })
		},
	},
	"attr*_value": {
		"Work attribute's value",
		func(w *seed.Work, k, v string) error {
			return workAttribute(w, k, func(a *seed.WorkAttribute) error { return setString(&a.Value, v) })
		},
	},
	"disambiguation": {
		"Comment disambiguating this work from others with similar names",
		func(w *seed.Work, k, v string) error { return setString(&w.Disambiguation, v) },
	},
	"edit_note": {
		"Note attached to edit",
		func(w *seed.Work, k, v string) error { return setString(&w.EditNote, v) },
	},
	"iswcs": {
		"Comma-separated ISWCs identifying work",
		func(w *seed.Work, k, v string) error { return setStringSlice(&w.ISWCs, v, ",") },
	},
	"languages": {
		"Comma-separated integer [language IDs](" + langIDURL + ") for work lyrics",
		func(w *seed.Work, k, v string) error {
			for _, s := range strings.Split(v, ",") {
				if d, err := strconv.Atoi(s); err != nil {
					return err
				} else {
					w.Languages = append(w.Languages, seed.Language(d))
				}
			}
			return nil
		},
	},
	"mbid": {
		"MBID of existing work to edit (if empty, create work)",
		func(w *seed.Work, k, v string) error { return setMBID(&w.MBID, v) },
	},
	"name": {
		"Work's name",
		func(w *seed.Work, k, v string) error { return setString(&w.Name, v) },
	},
	"type": {
		"Integer [work type](" + workTypeURL + ")",
		func(w *seed.Work, k, v string) error { return setInt((*int)(&w.Type), v) },
	},
}

func init() {
	// Add common fields.
	addRelationshipFields(workFields,
		func(fn relFunc) interface{} {
			return func(w *seed.Work, k, v string) error {
				return indexedField(&w.Relationships, k, "rel",
					func(rel *seed.Relationship) error { return fn(rel, k, v) })
			}
		})
	addURLFields(workFields,
		func(fn urlFunc) interface{} {
			return func(w *seed.Work, k, v string) error {
				return indexedField(&w.URLs, k, "url",
					func(url *seed.URL) error { return fn(url, v) })
			}
		})
}

// Helper function to make code for setting indexed fields slightly less horrendous.
func workAttribute(w *seed.Work, k string, fn func(*seed.WorkAttribute) error) error {
	return indexedField(&w.Attributes, k, "attr", fn)
}
