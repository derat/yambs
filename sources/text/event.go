// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// eventFields defines fields that can be set in a seed.Event.
var eventFields = map[string]fieldInfo{
	"begin_date": {
		`Date when event began as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(e *seed.Event, k, v string) error { return setDate(&e.BeginDate, v) },
	},
	// TODO: Also accept "canceled"?
	"cancelled": {
		`Whether event was cancelled ("1" or "true" if true)`,
		func(e *seed.Event, k, v string) error { return setBool(&e.Cancelled, v) },
	},
	"disambiguation": {
		"Comment disambiguating this event from others with similar names",
		func(e *seed.Event, k, v string) error { return setString(&e.Disambiguation, v) },
	},
	"edit_note": {
		"Note attached to edit",
		func(e *seed.Event, k, v string) error { return setString(&e.EditNote, v) },
	},
	"end_date": {
		`Date when event ended as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(e *seed.Event, k, v string) error { return setDate(&e.EndDate, v) },
	},
	"mbid": {
		"MBID of existing event to edit (if empty, create event)",
		func(e *seed.Event, k, v string) error { return setMBID(&e.MBID, v) },
	},
	"name": {
		"Event's name",
		func(e *seed.Event, k, v string) error { return setString(&e.Name, v) },
	},
	"setlist": {
		"Event's setlist",
		func(e *seed.Event, k, v string) error { return setString(&e.Setlist, v) },
	},
	"time": {
		`Event's starting time as "HH:MM"`,
		func(e *seed.Event, k, v string) error { return setString(&e.Time, v) },
	},
	"type": {
		"Integer [event type](" + eventTypeURL + ")",
		func(e *seed.Event, k, v string) error { return setInt((*int)(&e.Type), v) },
	},
}

func init() {
	// Add common fields.
	addRelationshipFields(eventFields,
		func(fn relFunc) interface{} {
			return func(e *seed.Event, k, v string) error {
				return indexedField(&e.Relationships, k, "rel",
					func(rel *seed.Relationship) error { return fn(rel, k, v) })
			}
		})
	addURLFields(eventFields,
		func(fn urlFunc) interface{} {
			return func(e *seed.Event, k, v string) error {
				return indexedField(&e.URLs, k, "url",
					func(url *seed.URL) error { return fn(url, v) })
			}
		})
}
