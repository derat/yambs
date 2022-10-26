// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"fmt"
	"net/url"
	"strings"
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

func (rec *Recording) Type() Type { return RecordingType }

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
