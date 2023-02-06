// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/derat/yambs/db"
)

// Recording holds data used to seed the "Add Standalone Recording" form at
// https://musicbrainz.org/recording/create and the edit-recording form at
// https://musicbrainz.org/recording/<MBID>/edit.
// See https://musicbrainz.org/doc/Recording for more information about recording entities.
type Recording struct {
	// MBID contains the recording's MBID (for editing an existing recording rather than
	// creating a new one).
	MBID string
	// Name contains the recording's title.
	Name string
	// Artist contains the MBID of the artist primarily credited with the recording.
	// TODO: Drop this in favor of only using Artists?
	Artist string
	// Artists contains detailed information about artists credited with the recording.
	Artists []ArtistCredit
	// Disambiguation differentiates this recording from other recordings with similar names.
	// See https://musicbrainz.org/doc/Disambiguation_Comment.
	Disambiguation string
	// Length contains the recording's duration.
	Length time.Duration
	// Video is true if this is a video recording.
	// Per https://musicbrainz.org/doc/How_to_Add_Standalone_Recordings, "an audio track uploaded to
	// Youtube with a static photo does not qualify as a video, this should be used only for actual
	// videos".
	Video bool
	// ISRCs contains 12-byte alphanumeric codes that identify audio or music video recordings.
	// See https://musicbrainz.org/doc/ISRC.
	ISRCs []string
	// URLs contains relationships between this recording and one or more URLs.
	// See https://musicbrainz.org/doc/Style/Relationships/URLs.
	//
	// As of 20221107, https://musicbrainz.org/recording/create lists the following types:
	//
	//  LinkType_License_Recording_URL ("license")
	//  LinkType_PurchaseForDownload_Recording_URL ("purchase for download")
	//  LinkType_DownloadForFree_Recording_URL ("download for free")
	//  LinkType_StreamingMusic_Recording_URL ("stream for free")
	//  LinkType_StreamingPaid_Recording_URL ("streaming page")
	//  LinkType_Crowdfunding_Recording_URL ("crowdfunding page")
	URLs []URL
	// Relationships contains (non-URL) relationships between this recording and other entities.
	Relationships []Relationship
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
}

func (rec *Recording) Entity() Entity { return RecordingEntity }

func (rec *Recording) Description() string {
	var parts []string
	if rec.MBID != "" {
		parts = append(parts, truncate(rec.MBID, mbidPrefixLen, false))
	}
	if rec.Name != "" {
		parts = append(parts, truncate(rec.Name, maxDescLen, true))
	}
	if s := artistCreditsDesc(rec.Artists); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "[unknown]"
	}
	return strings.Join(parts, " / ")
}

func (rec *Recording) URL(serverURL string) string {
	if rec.MBID != "" {
		return serverURL + "/recording/" + rec.MBID + "/edit"
	}
	return serverURL + "/recording/create"
}

func (rec *Recording) Params() url.Values {
	vals := make(url.Values)
	if rec.Name != "" {
		vals.Set("edit-recording.name", rec.Name)
	}
	if rec.Artist != "" {
		vals.Set("artist", rec.Artist)
	}
	for i, ac := range rec.Artists {
		ac.setParams(vals, fmt.Sprintf("edit-recording.artist_credit.names.%d.", i))
	}
	if rec.Disambiguation != "" {
		vals.Set("edit-recording.comment", rec.Disambiguation)
	}
	if rec.Length != 0 {
		vals.Set("edit-recording.length", fmt.Sprintf("%d", rec.Length.Milliseconds()))
	}
	if rec.Video {
		vals.Set("edit-recording.video", "1")
	}
	for i, isrc := range rec.ISRCs {
		vals.Set(fmt.Sprintf("edit-recording.isrcs.%d", i), isrc)
	}
	for i, u := range rec.URLs {
		u.setParams(vals, fmt.Sprintf("edit-recording.url.%d.", i), rec.Method())
	}
	for i, rel := range rec.Relationships {
		rel.setParams(vals, fmt.Sprintf("rels.%d.", i))
	}
	if rec.EditNote != "" {
		vals.Set("edit-recording.edit_note", rec.EditNote)
	}
	return vals
}

// I'm not sure how bogus it is to use GET instead of POST here,
// but there's no documentation about seeding recordings. ¯\_(ツ)_/¯
// When I tried using POST, the recording length wasn't accepted
// (maybe it needs to be MM:SS instead of milliseconds?), and it's
// generally more annoying to use POST since there's an interstitial
// page to prevent XSRFs that the user needs to click through.
func (rec *Recording) Method() string { return http.MethodGet }

func (rec *Recording) Finish(ctx context.Context, db *db.DB) error {
	for i := range rec.Artists {
		ac := &rec.Artists[i]
		if ac.MBID != "" {
			var err error
			if ac.ID, err = db.GetDatabaseID(ctx, ac.MBID); err != nil {
				return err
			}
			ac.MBID = ""
		}
	}
	return nil
}
