// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package seed generates URLs that pre-fill fields when adding entities to MusicBrainz.
package seed

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxDescLen    = 40 // max length for description components
	mbidPrefixLen = 8
)

// Edit represents a seeded MusicBrainz edit.
type Edit interface {
	// Description returns a human-readable description of the edit.
	Description() string
	// URL returns a URL to seed the edit form.
	URL() string
	// Params returns form values that should be sent to seed the edit form.
	Params() url.Values
	// CanGet() returns true if the request for URL can use the GET method rather than POST.
	// GET is preferable since it avoids an anti-CSRF interstitial page.
	CanGet() bool
}

// Recording holds data used to seed the "Add Standalone Recording" form at
// https://musicbrainz.org/recording/create and the edit-recording form at
// https://musicbrainz.org/recording/<MBID>/edit.
// See https://musicbrainz.org/doc/Recording for more information about recording entities.
type Recording struct {
	// MBID contains the recording's MBID (for editing an existing recording rather than
	// creating a new one).
	MBID string
	// Title contains the recording's title.
	Title string
	// Artist contains the MBID of the artist primarily credited with the recording.
	// TODO: Drop this in favor of only using ArtistCredits?
	Artist string
	// ArtistCredits contains detailed information about artists credited with the recording.
	ArtistCredits []ArtistCredit
	// Length contains the recording's duration.
	Length time.Duration
	// Video is true if this is a video recording.
	// Per https://musicbrainz.org/doc/How_to_Add_Standalone_Recordings, "an audio track uploaded to
	// Youtube with a static photo does not qualify as a video, this should be used only for actual
	// videos".
	Video bool
	// Disambiguation differentiates this recording from other recordings with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// ISRCs contains 12-byte alphanumeric codes that identify audio or music video recordings.
	// See https://musicbrainz.org/doc/ISRC.
	ISRCs []string
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
	// TODO: Figure out if there's any way to seed relationships or external links
	// for this form. Per https://community.metabrainz.org/t/seeding-recordings/188972/12?u=derat,
	// I couldn't find one. Is it possible to do this through a separate edit?
}

func (rec *Recording) Description() string {
	var parts []string
	if rec.MBID != "" {
		parts = append(parts, truncate(rec.MBID, mbidPrefixLen, false))
	}
	if rec.Title != "" {
		parts = append(parts, truncate(rec.Title, maxDescLen, true))
	}
	if len(rec.ArtistCredits) > 0 {
		var s string
		for _, ac := range rec.ArtistCredits {
			if ac.NameAsCredited != "" {
				s += ac.NameAsCredited
			} else if ac.Name != "" {
				s += ac.Name
			} else if ac.MBID != "" {
				s += truncate(ac.MBID, mbidPrefixLen, false)
			} else {
				continue
			}
			s += ac.JoinPhrase
		}
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (rec *Recording) URL() string {
	if rec.MBID != "" {
		return "https://musicbrainz.org/recording/" + rec.MBID + "/edit"
	}
	return "https://musicbrainz.org/recording/create"
}

func (rec *Recording) Params() url.Values {
	vals := make(url.Values)
	if rec.Title != "" {
		vals.Set("edit-recording.name", rec.Title)
	}
	if rec.Artist != "" {
		vals.Set("artist", rec.Artist)
	}
	for i, ac := range rec.ArtistCredits {
		ac.setParams(vals, fmt.Sprintf("edit-recording.artist_credit.names.%d.", i))
	}
	if rec.Length != 0 {
		vals.Set("edit-recording.length", fmt.Sprintf("%d", rec.Length.Milliseconds()))
	}
	if rec.Video {
		vals.Set("edit-recording.video", "1")
	}
	if rec.Disambiguation != "" {
		vals.Set("edit-recording.comment", rec.Disambiguation)
	}
	for i, isrc := range rec.ISRCs {
		vals.Set(fmt.Sprintf("edit-recording.isrcs.%d", i), isrc)
	}
	if rec.EditNote != "" {
		vals.Set("edit-recording.edit_note", rec.EditNote)
	}
	return vals
}

func (rec *Recording) CanGet() bool { return true }

// ArtistCredit holds detailed information about a credited artist.
type ArtistCredit struct {
	// MBID contains the artist entity's MBID, if known.
	// This annoyingly doesn't seem to work for the /recording/create form,
	// so set ID instead in that case (see db.GetArtistID).
	MBID string
	// ID contains the artist's database ID (i.e. the 'id' column from the 'artist' table).
	// This is only needed for the /recording/create form, I think.
	ID int32
	// Name contains the artist's name. This is unneeded if MBID or ID is set.
	Name string
	// NameAsCredited contains the name under which the artist was credited.
	// This is only needed if it's different than MBID or Name.
	NameAsCredited string
	// JoinPhrase contains text for joining this artist's name with the next one's, e.g. " & ".
	JoinPhrase string
}

// setParams sets query parameters in vals corresponding to non-empty fields in ac.
// The supplied prefix (e.g. "artist_credit.names.0.") is prepended before each parameter name.
func (ac *ArtistCredit) setParams(vals url.Values, prefix string) {
	var id string
	if ac.ID > 0 {
		id = fmt.Sprint(ac.ID)
	}
	for k, v := range map[string]string{
		"artist.id":   id,
		"mbid":        ac.MBID,
		"artist.name": ac.Name,
		"name":        ac.NameAsCredited,
		"join_phrase": ac.JoinPhrase,
	} {
		if v != "" {
			vals.Set(prefix+k, v)
		}
	}
}

// Release holds data used to seed the "Add Release" form at http://musicbrainz.org/release/add.
// See https://musicbrainz.org/doc/Release for more information about releases and
// https://wiki.musicbrainz.org/Development/Release_Editor_Seeding for information about seeding
// this form.
type Release struct {
	// Title contains the release's title.
	Title string
	// Artist contains the name of the artist primarily credited with the release.
	Artist string
	// Date contains the date on which the release was issued.
	Date time.Time
	// TODO: Add a zillion other fields.
}

func (rel *Release) Description() string { return rel.Title + " / " + rel.Artist }

func (rel *Release) URL() string { return "https://musicbrainz.org/release/add" }

func (rel *Release) Params() url.Values {
	vals := make(url.Values)
	if rel.Title != "" {
		vals.Set("name", rel.Title)
	}
	if rel.Artist != "" {
		vals.Set("artist_credit.names.0.name", rel.Artist)
	}
	if !rel.Date.IsZero() {
		vals.Set("events.0.date.year", strconv.Itoa(rel.Date.Year()))
		vals.Set("events.0.date.month", strconv.Itoa(int(rel.Date.Month())))
		vals.Set("events.0.date.day", strconv.Itoa(rel.Date.Day()))
	}
	return vals
}

func (rel *Release) CanGet() bool { return false }

func truncate(orig string, max int, ellide bool) string {
	if len(orig) <= max {
		return orig
	}
	if ellide {
		return orig[:max-1] + "…"
	}
	return orig[:max]
}
