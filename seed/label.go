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

// Label holds data used to seed the "Add Label" form at https://musicbrainz.org/label/create
// and the edit-label form at https://musicbrainz.org/label/<MBID>/edit.
// See https://musicbrainz.org/doc/Label for more information about label entities.
type Label struct {
	// MBID contains the label's MBID (for editing an existing label rather than creating a new one).
	MBID string
	// Name contains the label's name.
	Name string
	// Disambiguation differentiates this label from other label with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// Type describes the label's main activity.
	// See https://musicbrainz.org/doc/Label/Type.
	Type LabelType
	// AreaName is used to fill the search field for the label's area of origin.
	// TODO: Find some way to seed by MBID. There are hidden "edit-label.area.gid" and
	// "edit-label.area_id" inputs in the form, and there are a few references to the latter in test
	// code in the musicbrainz-server, but I haven't managed to fill the field by passing MBIDs or
	// database IDs via either parameter. The field oddly still turns green, though.
	AreaName string
	// LabelCode contains the 4- or 5-digit label code (i.e. without the "LC-" prefix, and with or
	// without leading zeros). See https://musicbrainz.org/doc/Label/Label_Code.
	LabelCode string
	// IPICodes contains the label's Interested Party Information code(s) assigned by the CISAC database
	// for musical rights management. See https://musicbrainz.org/doc/IPI.
	IPICodes []string
	// ISNICodes contains the label's International Standard Name Identifier(s).
	// See https://musicbrainz.org/doc/ISNI.
	ISNICodes []string
	// BeginYear contains the year when the label started, or 0 if unknown.
	BeginYear int
	// BeginMonth contains the 1-indexed month when the label started, or 0 if unknown.
	BeginMonth int
	// BeginDay contains the day of the month when the label started, or 0 if unknown.
	BeginDay int
	// EndYear contains the year when the label ended, or 0 if unknown.
	EndYear int
	// EndMonth contains the 1-indexed month when the label ended, or 0 if unknown.
	EndMonth int
	// EndDay contains the day of the month when the label ended, or 0 if unknown.
	EndDay int
	// Ended describes whether the label has ended.
	Ended bool
	// Relationships contains (non-URL) relationships between this label and other entities.
	Relationships []Relationship
	// URLs contains relationships between this label and one or more URLs.
	// See https://musicbrainz.org/doc/Style/Relationships/URLs.
	URLs []URL
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
}

func (l *Label) Entity() Entity { return LabelEntity }

func (l *Label) Description() string {
	var parts []string
	if l.MBID != "" {
		parts = append(parts, truncate(l.MBID, mbidPrefixLen, false))
	}
	if l.Name != "" {
		parts = append(parts, l.Name)
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (l *Label) URL(srv string) string {
	if l.MBID != "" {
		return "https://" + srv + "/label/" + l.MBID + "/edit"
	}
	return "https://" + srv + "/label/create"
}

func (l *Label) Params() url.Values {
	vals := make(url.Values)
	if l.Name != "" {
		vals.Set("edit-label.name", l.Name)
	}
	if l.Disambiguation != "" {
		vals.Set("edit-label.comment", l.Disambiguation)
	}
	if l.Type != 0 {
		vals.Set("edit-label.type_id", strconv.Itoa(int(l.Type)))
	}
	if l.AreaName != "" {
		vals.Set("edit-label.area.name", l.AreaName)
	}
	if l.LabelCode != "" {
		vals.Set("edit-label.label_code", l.LabelCode)
	}
	for i, code := range l.IPICodes {
		vals.Set(fmt.Sprintf("edit-label.ipi_codes.%d", i), code)
	}
	for i, code := range l.ISNICodes {
		vals.Set(fmt.Sprintf("edit-label.isni_codes.%d", i), code)
	}

	// It'd be nice if labels could share date-setting code with the similar-looking
	// begin/end/ended date ranges for relationships, but the seeding mechanism seems
	// completely different: relationships use "begin_date" and "end_date" parameters
	// with "YYYY-MM-DD" values, while labels have separate "period.begin_date.year",
	// "period.begin_date.month", etc. parameters.
	for _, p := range []struct {
		name string
		val  int
	}{
		{"begin_date.year", l.BeginYear},
		{"begin_date.month", l.BeginMonth},
		{"begin_date.day", l.BeginDay},
		{"end_date.year", l.EndYear},
		{"end_date.month", l.EndMonth},
		{"end_date.day", l.EndDay},
	} {
		if p.val > 0 {
			vals.Set("edit-label.period."+p.name, strconv.Itoa(p.val))
		}
	}
	if l.Ended {
		vals.Set("edit-label.period.ended", "1")
	}

	for i, rel := range l.Relationships {
		rel.setParams(vals, fmt.Sprintf("rels.%d.", i))
	}
	for i, u := range l.URLs {
		u.setParams(vals, fmt.Sprintf("edit-label.url.%d.", i), l.Method())
	}
	if l.EditNote != "" {
		vals.Set("edit-label.edit_note", l.EditNote)
	}
	return vals
}

func (l *Label) Method() string { return http.MethodGet }

func (l *Label) Finish(ctx context.Context, db *db.DB) error { return nil }
