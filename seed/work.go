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

	"github.com/derat/yambs/db"
)

// Work holds data used to seed the "Add Work" form at https://musicbrainz.org/work/create
// and the edit-work form at https://musicbrainz.org/work/<MBID>/edit.
// See https://musicbrainz.org/doc/Work for more information about work entities.
type Work struct {
	// MBID contains the work's MBID (for editing an existing work rather than creating a new one).
	MBID string
	// Name contains the work's name.
	Name string
	// Disambiguation differentiates this work from other works with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// Type describes the work's type.
	// See "Types of works" at https://musicbrainz.org/doc/Work.
	Type WorkType
	// Languages contains database IDs corresponding to the language(s) of the work's lyrics.
	Languages []Language
	// ISWCs contains unique identifiers for the work in T-DDD.DDD.DDD-C format.
	// See https://wiki.musicbrainz.org/ISWC.
	ISWCs []string
	// Attributes contains attributes describing this work.
	Attributes []WorkAttribute
	// Relationships contains (non-URL) relationships between this work and other entities.
	Relationships []Relationship
	// URLs contains relationships between this work and one or more URLs.
	// See https://musicbrainz.org/doc/Style/Relationships/URLs.
	URLs []URL
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
}

func (w *Work) Entity() Entity { return WorkEntity }

func (w *Work) Description() string {
	var parts []string
	if w.MBID != "" {
		parts = append(parts, truncate(w.MBID, mbidPrefixLen, false))
	}
	if w.Name != "" {
		parts = append(parts, w.Name)
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (w *Work) URL(srv string) string {
	if w.MBID != "" {
		return "https://" + srv + "/work/" + w.MBID + "/edit"
	}
	return "https://" + srv + "/work/create"
}

func (w *Work) Params() url.Values {
	// I haven't found any documentation about seeding works, but the form seems
	// to function similarly to the recording form.
	vals := make(url.Values)
	if w.Name != "" {
		vals.Set("edit-work.name", w.Name)
	}
	if w.Disambiguation != "" {
		vals.Set("edit-work.comment", w.Disambiguation)
	}
	if w.Type != 0 {
		vals.Set("edit-work.type_id", strconv.Itoa(int(w.Type)))
	}
	for i, lang := range w.Languages {
		vals.Set(fmt.Sprintf("edit-work.languages.%d", i), strconv.Itoa(int(lang)))
	}
	for i, iswc := range w.ISWCs {
		vals.Set(fmt.Sprintf("edit-work.iswcs.%d", i), iswc)
	}
	for i, attr := range w.Attributes {
		attr.setParams(vals, fmt.Sprintf("edit-work.attributes.%d.", i))
	}
	for i, rel := range w.Relationships {
		rel.setParams(vals, fmt.Sprintf("rels.%d.", i))
	}
	for i, u := range w.URLs {
		u.setParams(vals, fmt.Sprintf("edit-work.url.%d.", i), w.Method())
	}
	if w.EditNote != "" {
		vals.Set("edit-work.edit_note", w.EditNote)
	}
	return vals
}

func (w *Work) Method() string { return http.MethodGet }

func (w *Work) Finish(ctx context.Context, db *db.DB) error { return nil }

// WorkAttribute describes an attribute associated with a work.
type WorkAttribute struct {
	// Type specifies the attribute's type.
	Type WorkAttributeType
	// Value holds the attribute's value, e.g. an actual ID.
	Value string
}

// setParams sets query parameters in vals corresponding to non-empty fields in attr.
// The supplied prefix (e.g. "edit-work.attributes.0.") is prepended before each parameter name.
func (attr *WorkAttribute) setParams(vals url.Values, prefix string) {
	if attr.Type != 0 {
		vals.Set(prefix+"type_id", strconv.Itoa(int(attr.Type)))
	}
	if attr.Value != "" {
		vals.Set(prefix+"value", attr.Value)
	}
}
