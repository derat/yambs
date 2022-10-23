// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package seed generates URLs that pre-fill fields when adding entities to MusicBrainz.
package seed

import (
	"fmt"
	"net/url"
	"time"
)

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

// URL returns a URL to seed the "Add Standalone Recording" or edit-recording form via a GET request.
func (rec *Recording) URL() string {
	us := "https://musicbrainz.org/recording/create"
	if rec.MBID != "" {
		us = "https://musicbrainz.org/recording/" + rec.MBID + "/edit"
	}
	u, _ := url.Parse(us)

	vals := make(url.Values)
	rec.SetParams(vals)
	u.RawQuery = vals.Encode()
	return u.String()
}

// SetParams sets query parameters in vals corresponding to non-empty fields in rec.
func (rec *Recording) SetParams(vals url.Values) {
	if rec.Title != "" {
		vals.Set("edit-recording.name", rec.Title)
	}
	if rec.Artist != "" {
		vals.Set("artist", rec.Artist)
	}
	for i, ac := range rec.ArtistCredits {
		ac.SetParams(vals, fmt.Sprintf("edit-recording.artist_credit.names.%d.", i))
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
}

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

// SetParams sets query parameters in vals corresponding to non-empty fields in ac.
// The supplied prefix (e.g. "artist_credit.names.0.") is prepended before each parameter name.
func (ac *ArtistCredit) SetParams(vals url.Values, prefix string) {
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
