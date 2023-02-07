// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/derat/yambs/mbdb"
)

// Release holds data used to seed the "Add Release" form at http://musicbrainz.org/release/add.
// See https://musicbrainz.org/doc/Release for more information about releases and
// https://wiki.musicbrainz.org/Development/Release_Editor_Seeding for information about seeding
// this form.
type Release struct {
	// MBID contains the release's MBID (for editing an existing release rather than creating a new one).
	MBID string
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
	//  LinkType_FreeStreaming_Release_URL ("stream for free")
	//  LinkType_DiscographyEntry_Release_URL ("discography entry")
	//  LinkType_License_Release_URL ("license")
	//  LinkType_ShowNotes_Release_URL ("show notes")
	//  LinkType_Crowdfunding_Release_URL ("crowdfunding page")
	//  LinkType_Streaming_Release_URL ("streaming page")
	URLs []URL
	// EditNote contains the note attached to the edit.
	// See https://musicbrainz.org/doc/Edit_Note.
	EditNote string
	// RedirectURI contains a URL for MusicBrainz to redirect to after the edit is created.
	// The MusicBrainz server will add a "release_mbid" query parameter containing the
	// new release's MBID.
	RedirectURI string
}

func (rel *Release) Entity() Entity { return ReleaseEntity }

func (rel *Release) Description() string {
	var parts []string
	if rel.MBID != "" {
		parts = append(parts, truncate(rel.MBID, mbidPrefixLen, false))
	}
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

func (rel *Release) URL(serverURL string) string {
	if rel.MBID != "" {
		return serverURL + "/release/" + rel.MBID + "/edit"
	}
	return serverURL + "/release/add"
}

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
		u.setParams(vals, fmt.Sprintf("urls.%d.", i), rel.Method())
	}
	set("edit_note", rel.EditNote)
	set("redirect_uri", rel.RedirectURI)
	return vals
}

func (rel *Release) Method() string { return http.MethodPost }

func (rel *Release) Finish(ctx context.Context, db *mbdb.DB) error { return nil }

// Autofill attempts to automatically fill empty fields in rel.
// The Language and Script fields are filled based on the release and track titles.
// If network is true, network requests may be made.
func (rel *Release) Autofill(ctx context.Context, network bool) {
	if rel.Language == "" || rel.Script == "" {
		titles := []string{rel.Title}
		for _, med := range rel.Mediums {
			for _, track := range med.Tracks {
				titles = append(titles, track.Title)
			}
		}
		// Try to determine the language and script via an API call if allowed.
		if network {
			if lang, script, err := detectLangNetwork(ctx, titles); err != nil {
				log.Print("Detecting language via network failed: ", err)
			} else {
				if rel.Language == "" {
					rel.Language = lang
				}
				if rel.Script == "" {
					rel.Script = script
				}
			}
		}
		// Fall back to detecting the script locally.
		if rel.Script == "" {
			rel.Script = detectScriptLocal(titles)
		}
	}

	// Try to guess the release group type if it isn't already set.
	// See https://musicbrainz.org/doc/Release_Group/Type.
	if len(rel.Types) == 0 {
		var numTracks int
		var totalLen time.Duration
		relTitle := removeExtraTitleInfo(rel.Title)
		trackMatchesRelease := false  // a track shares the release's title
		tracksAllSingleLength := true // all tracks are <= maxSingleTrackLen
		for _, med := range rel.Mediums {
			for _, tr := range med.Tracks {
				numTracks++
				totalLen += tr.Length
				if title := removeExtraTitleInfo(tr.Title); title == relTitle {
					trackMatchesRelease = true
				}
				if tr.Length > maxSingleTrackLen {
					tracksAllSingleLength = false
				}
			}
		}

		switch {
		case strings.HasSuffix(relTitle, " EP"):
			// Only classify the release as an EP if its name ends with "EP".
			rel.Types = append(rel.Types, ReleaseGroupType_EP)
		case tracksAllSingleLength &&
			(numTracks == 1 || (numTracks <= maxSingleTracks && trackMatchesRelease)):
			// Classify the release as a single if none of the tracks are too long and there's
			// either a single track or a few tracks, one of which has the same name as the release.
			rel.Types = append(rel.Types, ReleaseGroupType_Single)
		case totalLen >= minAlbumLen || numTracks >= minAlbumTracks:
			// Fall back to calling it an album.
			rel.Types = append(rel.Types, ReleaseGroupType_Album)
		}
	}
}

const (
	maxSingleTracks   = 4                // max tracks in singles
	maxSingleTrackLen = 15 * time.Minute // max length for tracks in singles
	minAlbumLen       = 40 * time.Minute // min total length for release to be an album
	minAlbumTracks    = 8                // min tracks in album
)

// extraTitleInfoRegexp matches a trailing string like " (explicit)" or " [explicit]".
var extraTitleInfoRegexp = regexp.MustCompile(`\s+(\([^)]+\)|\[[^]]+\])$`)

// removeExtraTitleInfo removes a suffix like " (explicit)" or " [explicit]" from title.
// Note that this may also remove a non-ETI portion of the title,
// e.g. "Don't You (Forget About Me)".
func removeExtraTitleInfo(title string) string {
	return title[:len(title)-len(extraTitleInfoRegexp.FindString(title))]
}

// AddCoverArtRedirectURI can be used as a Release's RedirectURI to automatically redirect to the
// "Add Cover Art" page after the release has been created.
//
// Regrettably, the Add Release page passes the MBID to the redirect URL via a "release_mbid" query
// parameter, while the Add Cover Art form requires the MBID to be passed as part of the path
// (https://musicbrainz.org/release/<mbid>/add-cover-art).
//
// TODO: Change this to not redirect through yambsd if/when the MB server provides a way to rewrite
// the final redirect URL.
const AddCoverArtRedirectURI = "https://yambs.erat.org/redirect-add-cover-art"

// NewAddCoverArtEdit returns an informational edit linking to mbid's add-cover-art page.
func NewAddCoverArtEdit(desc, mbid string) (*Info, error) {
	return NewInfo(desc, "/release/"+mbid+"/add-cover-art")
}

// ReleaseEvent contains an event corresponding to a release. Unknown fields can be omitted.
type ReleaseEvent struct {
	// Date contains the event's date.
	Date Date
	// Country contains the event's country as an ISO code (e.g. "GB", "US", "FR").
	// "XW" corresponds to "[Worldwide]".
	Country string
}

// setParams sets query parameters in vals corresponding to non-empty fields in ev.
// The supplied prefix (e.g. "events.0.") is prepended before each parameter name.
func (ev *ReleaseEvent) setParams(vals url.Values, prefix string) {
	ev.Date.setParams(vals, prefix+"date.")
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
		ac.setParams(vals, prefix+fmt.Sprintf("artist_credit.names.%d.", i))
	}
}
