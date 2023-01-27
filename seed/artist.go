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

// Artist holds data used to seed the "Add Artist" form at https://musicbrainz.org/artist/create
// and the edit-artist form at https://musicbrainz.org/artist/<MBID>/edit.
// See https://musicbrainz.org/doc/Artist for more information about artist entities.
type Artist struct {
	// MBID contains the artist's MBID (for editing an existing artist rather than creating a new one).
	MBID string
	// Name contains the artist's official name.
	Name string
	// SortName contains a variant of the artist's name that should be used for sorting.
	// See https://musicbrainz.org/doc/Style/Artist/Sort_Name.
	SortName string
	// Disambiguation differentiates this artist from other artist with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// Type describes whether the artist is a person, group, or something else.
	Type ArtistType
	// Gender describes how a person or character identifies. Groups do not have genders.
	Gender Gender
	// AreaName is used to fill the search field for the area with which the artist primarily
	// identifies.
	AreaName string
	// IPICodes contains the artist's Interested Party Information code(s) assigned by the CISAC database
	// for musical rights management. See https://musicbrainz.org/doc/IPI.
	IPICodes []string
	// ISNICodes contains the artist's International Standard Name Identifier(s).
	// See https://musicbrainz.org/doc/ISNI.
	ISNICodes []string
	// BeginDate contains the date when the artist started.
	// For a person, this is the date of birth.
	// For a group, this is when the group was first formed.
	// For a character, this is when the character concept was created.
	BeginDate Date
	// BeginAreaName is used to fill the search field for the area where the artist started.
	BeginAreaName string
	// EndDate contains the date when the artist ended.
	// For a person, this is the date of death.
	// For a group, this is when the group was last dissolved.
	// For a character, this should not be set.
	EndDate Date
	// Ended describes whether the artist has ended.
	Ended bool
	// EndAreaName is used to fill the search field for the area where the artist ended.
	EndAreaName string
	// Relationships contains (non-URL) relationships between this artist and other entities.
	Relationships []Relationship
	// URLs contains relationships between this artist and one or more URLs.
	// See https://musicbrainz.org/doc/Style/Relationships/URLs.
	URLs []URL
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
}

func (a *Artist) Entity() Entity { return ArtistEntity }

func (a *Artist) Description() string {
	var parts []string
	if a.MBID != "" {
		parts = append(parts, truncate(a.MBID, mbidPrefixLen, false))
	}
	if a.Name != "" {
		parts = append(parts, a.Name)
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (a *Artist) URL(srv string) string {
	if a.MBID != "" {
		return "https://" + srv + "/artist/" + a.MBID + "/edit"
	}
	return "https://" + srv + "/artist/create"
}

func (a *Artist) Params() url.Values {
	vals := make(url.Values)
	if a.Name != "" {
		vals.Set("edit-artist.name", a.Name)
	}
	if a.SortName != "" {
		vals.Set("edit-artist.sort_name", a.SortName)
	}
	if a.Disambiguation != "" {
		vals.Set("edit-artist.comment", a.Disambiguation)
	}
	if a.Type != 0 {
		vals.Set("edit-artist.type_id", strconv.Itoa(int(a.Type)))
	}
	if a.Gender != 0 {
		vals.Set("edit-artist.gender_id", strconv.Itoa(int(a.Gender)))
	}
	if a.AreaName != "" {
		vals.Set("edit-artist.area.name", a.AreaName)
	}
	for i, code := range a.IPICodes {
		vals.Set(fmt.Sprintf("edit-artist.ipi_codes.%d", i), code)
	}
	for i, code := range a.ISNICodes {
		vals.Set(fmt.Sprintf("edit-artist.isni_codes.%d", i), code)
	}

	a.BeginDate.setParams(vals, "edit-artist.period.begin_date.")
	if a.BeginAreaName != "" {
		vals.Set("edit-artist.begin_area.name", a.BeginAreaName)
	}
	a.EndDate.setParams(vals, "edit-artist.period.end_date.")
	if a.Ended {
		vals.Set("edit-artist.period.ended", "1")
	}
	if a.EndAreaName != "" {
		vals.Set("edit-artist.end_area.name", a.EndAreaName)
	}

	for i, rel := range a.Relationships {
		rel.setParams(vals, fmt.Sprintf("rels.%d.", i))
	}
	for i, u := range a.URLs {
		u.setParams(vals, fmt.Sprintf("edit-artist.url.%d.", i), a.Method())
	}
	if a.EditNote != "" {
		vals.Set("edit-artist.edit_note", a.EditNote)
	}
	return vals
}

func (a *Artist) Method() string { return http.MethodGet }

func (a *Artist) Finish(ctx context.Context, db *db.DB) error { return nil }
