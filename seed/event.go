// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/derat/yambs/mbdb"
)

// Event holds data used to seed the "Add Event" form at https://musicbrainz.org/event/create
// and the edit-event form at https://musicbrainz.org/event/<MBID>/edit.
// See https://musicbrainz.org/doc/Event for more information about event entities.
type Event struct {
	// MBID contains the event's MBID (for editing an existing event rather than creating a new one).
	MBID string
	// Name contains the event's name.
	Name string
	// Disambiguation differentiates this event from other events with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// Type describes the kind of that the event is.
	// See https://musicbrainz.org/doc/Event#Type.
	Type EventType
	// Cancelled is true if the event was cancelled (i.e. it did not take place).
	Cancelled bool
	// Setlist contains a list of the songs which were performed.
	// See https://musicbrainz.org/doc/Event/Setlist for details.
	Setlist string
	// BeginDate contains the date when the event started.
	BeginDate Date
	// EndDate contains the date when the ended ended.
	EndDate Date
	// Time contains the event's start time in "HH:MM" format.
	Time string
	// Relationships contains (non-URL) relationships between this event and other entities.
	Relationships []Relationship
	// URLs contains relationships between this event and one or more URLs.
	// See https://musicbrainz.org/doc/Style/Relationships/URLs.
	URLs []URL
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
}

func (e *Event) Entity() Entity { return EventEntity }

func (e *Event) Description() string {
	var parts []string
	if e.MBID != "" {
		parts = append(parts, truncate(e.MBID, mbidPrefixLen, false))
	}
	if e.Name != "" {
		parts = append(parts, e.Name)
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (e *Event) URL(serverURL string) string {
	if e.MBID != "" {
		return serverURL + "/event/" + e.MBID + "/edit"
	}
	return serverURL + "/event/create"
}

func (e *Event) Params() url.Values {
	vals := make(url.Values)
	if e.Name != "" {
		vals.Set("edit-event.name", e.Name)
	}
	if e.Disambiguation != "" {
		vals.Set("edit-event.comment", e.Disambiguation)
	}
	if e.Type != 0 {
		vals.Set("edit-event.type_id", strconv.Itoa(int(e.Type)))
	}
	if e.Cancelled {
		vals.Set("edit-event.cancelled", "1")
	}
	if e.Setlist != "" {
		vals.Set("edit-event.setlist", e.Setlist)
	}
	e.BeginDate.setParams(vals, "edit-event.period.begin_date.")
	e.EndDate.setParams(vals, "edit-event.period.end_date.")
	if e.Time != "" {
		vals.Set("edit-event.time", e.Time)
	}
	for i, rel := range e.Relationships {
		rel.setParams(vals, fmt.Sprintf("rels.%d.", i))
	}
	for i, u := range e.URLs {
		u.setParams(vals, fmt.Sprintf("edit-event.url.%d.", i), e.Method())
	}
	if e.EditNote != "" {
		vals.Set("edit-event.edit_note", e.EditNote)
	}
	return vals
}

func (e *Event) Method() string { return http.MethodGet }

func (e *Event) Finish(ctx context.Context, db *mbdb.DB) error { return nil }
