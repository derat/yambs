// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/db"
)

// Release holds data used to seed the "Add Release" form at http://musicbrainz.org/release/add.
// See https://musicbrainz.org/doc/Release for more information about releases and
// https://wiki.musicbrainz.org/Development/Release_Editor_Seeding for information about seeding
// this form.
type Release struct {
	// Title contains the release's title.
	Title string
	// ReleaseGroup the MBID of an existing release group.
	// See https://musicbrainz.org/doc/Release_Group.
	ReleaseGroup string
	// Types contains types for a new release group (if ReleaseGroup is empty).
	// See https://wiki.musicbrainz.org/Release_Group/Type.
	Types []ReleaseGroupType
	// Disambiguation differentiates this release from other releases with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// Annotation contains additional information that doesn't fit in MusicBrainz's data scheme.
	// See https://musicbrainz.org/doc/Annotation.
	Annotation string
	// Barcode contains the release's barcode. "none" indicates that the release has no barcode.
	Barcode string
	// Language contains the release's language as an ISO 639-3 code (e.g. "eng", "deu", "jpn").
	// See https://en.wikipedia.org/wiki/List_of_ISO_639-3_codes.
	Language string
	// Script contains the script of the text on the release as an ISO 15924 code (e.g. "Latn", "Cyrl").
	// See https://en.wikipedia.org/wiki/ISO_15924.
	Script string
	// Status contains the release's status.
	Status ReleaseStatus
	// Packaging contains the release's packaging as an English string.
	// See https://wiki.musicbrainz.org/Release/Packaging.
	Packaging ReleasePackaging
	// Events contains events corresponding to this release.
	Events []ReleaseEvent
	// Labels contains label-related information corresponding to this release.
	Labels []ReleaseLabel
	// ArtistCredits contains artists credited with the release.
	Artists []ArtistCredit
	// Mediums contains the release's media (which themselves contain tracklists).
	Mediums []Medium
	// URLs contains relationships between this release and one or more URLs.
	// See https://musicbrainz.org/doc/Style/Relationships/URLs.
	//
	// As of 20221028, https://musicbrainz.org/release/add lists the following types:
	//  LinkType_PurchaseForDownload_Release_URL ("purchase for download")
	//  LinkType_DownloadForFree_Release_URL ("download for free")
	//  LinkType_PurchaseForMailOrder_Release_URL ("purchase for mail-order")
	//  LinkType_StreamingMusic_Release_URL ("stream for free")
	//  LinkType_DiscographyEntry_Release_URL ("discography entry")
	//  LinkType_License_Release_URL ("license")
	//  LinkType_ShowNotes_Release_URL ("show notes")
	//  LinkType_Crowdfunding_Release_URL ("crowdfunding page")
	//  LinkType_StreamingPaid_Release_URL ("streaming page")
	URLs []URL
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
}

func (rel *Release) Type() Type { return ReleaseType }

func (rel *Release) Description() string {
	var parts []string
	if rel.Title != "" {
		parts = append(parts, rel.Title)
	}
	if s := artistCreditsDesc(rel.Artists); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (rel *Release) URL() string { return "https://musicbrainz.org/release/add" }

func (rel *Release) Params() url.Values {
	vals := make(url.Values)
	set := func(k, v string) {
		if v != "" {
			vals.Set(k, v)
		}
	}
	set("name", rel.Title)
	set("release_group", rel.ReleaseGroup)
	for _, t := range rel.Types {
		vals.Add("type", string(t))
	}
	set("comment", rel.Disambiguation)
	set("annotation", rel.Annotation)
	set("barcode", rel.Barcode)
	set("language", rel.Language)
	set("script", rel.Script)
	set("status", string(rel.Status))
	set("packaging", string(rel.Packaging))
	for i, ev := range rel.Events {
		ev.setParams(vals, fmt.Sprintf("events.%d.", i))
	}
	for i, rl := range rel.Labels {
		rl.setParams(vals, fmt.Sprintf("labels.%d.", i))
	}
	for i, ac := range rel.Artists {
		ac.setParams(vals, fmt.Sprintf("artist_credit.names.%d.", i))
	}
	for i, m := range rel.Mediums {
		m.setParams(vals, fmt.Sprintf("mediums.%d.", i))
	}
	for i, u := range rel.URLs {
		u.setParams(vals, fmt.Sprintf("urls.%d.", i))
	}
	set("edit_note", rel.EditNote)
	return vals
}

func (rel *Release) CanGet() bool { return false }

// ReleaseEvent contains an event corresponding to a release. Unknown fields can be omitted.
type ReleaseEvent struct {
	// Year contains the event's year, or 0 if unknown.
	Year int
	// Month contains the event's 1-indexed month, or 0 if unknown.
	Month int
	// Day contains the event's day, or 0 if unknown.
	Day int
	// Country contains the event's country as an ISO code (e.g. "GB", "US", "FR").
	// "XW" corresponds to "[Worldwide]".
	Country string
}

// setParams sets query parameters in vals corresponding to non-empty fields in ev.
// The supplied prefix (e.g. "events.0.") is prepended before each parameter name.
func (ev *ReleaseEvent) setParams(vals url.Values, prefix string) {
	if ev.Year > 0 {
		vals.Set(prefix+"date.year", strconv.Itoa(ev.Year))
	}
	if ev.Month > 0 {
		vals.Set(prefix+"date.month", strconv.Itoa(ev.Month))
	}
	if ev.Day > 0 {
		vals.Set(prefix+"date.day", strconv.Itoa(ev.Day))
	}
	if ev.Country != "" {
		vals.Set(prefix+"country", ev.Country)
	}
}

// ReleaseLabel contains label-related information associated with a release.
type ReleaseLabel struct {
	// MBID contains the label's MBID if known.
	MBID string
	// CatalogNumber contains the release's catalog number.
	CatalogNumber string
	// Name contains the label's name (to prefill the search field if MBID is empty).
	Name string
}

// setParams sets query parameters in vals corresponding to non-empty fields in rl.
// The supplied prefix (e.g. "labels.0.") is prepended before each parameter name.
func (rl *ReleaseLabel) setParams(vals url.Values, prefix string) {
	setParams(vals, map[string]string{
		"mbid":           rl.MBID,
		"catalog_number": rl.CatalogNumber,
		"name":           rl.Name,
	}, prefix)
}

// Medium describes a medium that is part of a release.
// See https://musicbrainz.org/doc/Medium.
type Medium struct {
	// Format contains the medium's format name.
	// See https://wiki.musicbrainz.org/Release/Format.
	Format MediumFormat
	// Name contains the medium's name (e.g. "Live & Unreleased").
	Name string
	// Tracks contains the medium's tracklist.
	Tracks []Track
	// TODO: Include position? It's inferred based on order, so maybe not.
}

// setParams sets query parameters in vals corresponding to non-empty fields in m.
// The supplied prefix (e.g. "mediums.0.") is prepended before each parameter name.
func (m *Medium) setParams(vals url.Values, prefix string) {
	setParams(vals, map[string]string{
		"format": string(m.Format),
		"name":   m.Name,
	}, prefix)

	for i, t := range m.Tracks {
		t.setParams(vals, prefix+fmt.Sprintf("track.%d.", i))
	}
}

// Track describes the way that a recording is represented on a medium.
// See https://musicbrainz.org/doc/Track.
type Track struct {
	// Title contains the track's name.
	Title string
	// Number contains a free-form track number.
	Number string
	// Recording contains the MBID of the recording corresponding to the track.
	Recording string
	// Length contains the track's duration.
	Length time.Duration
	// Artists contains the artists credited with the track.
	Artists []ArtistCredit
}

// setParams sets query parameters in vals corresponding to non-empty fields in tr.
// The supplied prefix (e.g. "mediums.0.tracks.5.") is prepended before each parameter name.
func (tr *Track) setParams(vals url.Values, prefix string) {
	setParams(vals, map[string]string{
		"name":      tr.Title,
		"number":    tr.Number,
		"recording": tr.Recording,
	}, prefix)

	if tr.Length != 0 {
		vals.Set(prefix+"length", fmt.Sprintf("%d", tr.Length.Milliseconds()))
	}
	for i, ac := range tr.Artists {
		ac.setParams(vals, prefix+fmt.Sprintf("artist_credit.%d.", i))
	}
}

func (rel *Release) Finish(ctx context.Context, db *db.DB) error { return nil }
