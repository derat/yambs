// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package bandcamp extracts information from Bandcamp pages.
package bandcamp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online/internal"
	"github.com/derat/yambs/web"
	"golang.org/x/net/html"
)

const (
	// finishTime is reserved to finish creating edits after querying the MusicBrainz API.
	finishTime = 3 * time.Second
)

// Provider implements internal.Provider for Bandcamp.
type Provider struct{}

// Release extracts release information from the supplied Bandcamp page.
// This is heavily based on the bandcamp_importer.user.js userscript:
// https://github.com/murdos/musicbrainz-userscripts/blob/master/bandcamp_importer.user.js
func (p *Provider) Release(ctx context.Context, page *web.Page, pageURL string,
	db *mbdb.DB, cfg *internal.Config) (rel *seed.Release, img *seed.Info, err error) {
	// Upgrade the scheme for later usage.
	if strings.HasPrefix(pageURL, "http://") {
		pageURL = "https" + pageURL[4:]
	}

	var album albumData
	if err := unmarshalAttr(page, "script[data-tralbum]", "data-tralbum", &album); err != nil {
		return nil, nil, fmt.Errorf("album data: %v", err)
	}
	var embed embedData
	if err := unmarshalAttr(page, "script[data-embed]", "data-embed", &embed); err != nil {
		return nil, nil, fmt.Errorf("embed data: %v", err)
	}

	rel = &seed.Release{
		Title: album.Current.Title,
		// TODO: Add logic for detecting "various artists", maybe.
		// The userscript checks if all tracks have titles like "artist - tracktitle" with
		// non-numeric artists (which would instead be a track number) and tests the album
		// artist against '^various(?: artists)?$'.
		// TODO: Consider passing this to parseArtists. There are a bunch of group names that
		// would be incorrectly split, though, so maybe not.
		Artists:   []seed.ArtistCredit{{Name: album.Artist}},
		Status:    seed.ReleaseStatus_Official,
		Packaging: seed.ReleasePackaging_None,
		Mediums:   []seed.Medium{seed.Medium{Format: seed.MediumFormat_DigitalMedia}},
	}

	if album.Current.Type == "track" && embed.AlbumEmbedData.Linkback != "" {
		return nil, nil, errors.New("track is part of " + embed.AlbumEmbedData.Linkback)
	}

	// Use a shortened context for querying MusicBrainz MBIDs so we'll have a bit of time left to
	// finish creating the edit even if we need to look up a bunch of different artists:
	// https://github.com/derat/yambs/issues/19
	shortCtx, shortCancel := mbdb.ShortenContext(ctx, finishTime)
	defer shortCancel()

	// Try to find the artist's MBID from the URL.
	baseURL := getBaseURL(pageURL)
	if baseURL != "" {
		rel.Artists[0].MBID = internal.GetArtistMBIDFromURL(shortCtx, db, baseURL, album.Artist)
	}

	// I'm guessing that the publish date is when the album was created in Bandcamp,
	// while the release date is when it was actually made available to users (but
	// can maybe also be set to some arbitrary date?). Follow the userscript's logic
	// of using the release date unless it precedes Bandcamp's launch.
	date := time.Time(album.Current.ReleaseDate)
	if date.IsZero() || date.Before(bandcampLaunch) {
		date = time.Time(album.Current.PublishDate)
	}
	if !date.IsZero() {
		rel.Events = []seed.ReleaseEvent{{
			Date:    seed.DateFromTime(date),
			Country: "XW",
		}}
	}

	// Add a single medium with all of the tracks.
	med := &rel.Mediums[0]
	var streamableTracks int
	artistPrefix := album.Artist + " - "
	for _, tr := range album.TrackInfo {
		track := seed.Track{Length: time.Duration(float64(time.Second) * tr.Duration)}
		if cfg.ExtractTrackArtists {
			track.Title, track.Artists = extractTrackArtists(tr.Title)
		} else if strings.HasPrefix(tr.Title, artistPrefix) {
			// Strip "Artist - " from the beginning of the track title even
			// if we weren't explicitly told to extract artist names:
			// https://github.com/derat/yambs/issues/16
			track.Title = tr.Title[len(artistPrefix):]
		} else {
			track.Title = tr.Title
		}
		med.Tracks = append(med.Tracks, track)

		if len(tr.File) != 0 {
			streamableTracks++
		}
	}

	// Look for hidden tracks. Apparently the count from the Open Graph description
	// indicates the number of tracks that will actually be included in the download.
	var metaTracks int
	if desc, err := page.Query(`meta[property="og:description"]`).Attr("content"); err == nil {
		if ms := metaDescRegexp.FindStringSubmatch(desc); len(ms) > 0 {
			metaTracks, _ = strconv.Atoi(ms[1])
		}
	}
	for i := len(med.Tracks); i < metaTracks; i++ {
		med.Tracks = append(med.Tracks, seed.Track{Title: "[unknown]"})
	}

	// Add URLs. This logic is lifted wholesale from the userscript.
	addURL := func(u string, lt seed.LinkType) {
		rel.URLs = append(rel.URLs, seed.URL{URL: u, LinkType: lt})
	}
	if pref := album.Current.DownloadPref; pref != 0 {
		if album.Current.FreeDownloadPage != "" ||
			pref == 1 ||
			(pref == 2 && album.Current.MinimumPrice == 0) {
			addURL(pageURL, seed.LinkType_DownloadForFree_Release_URL)
		}
		if pref == 2 {
			addURL(pageURL, seed.LinkType_PurchaseForDownload_Release_URL)
		}
	}
	if numTracks := len(med.Tracks); album.HasAudio && numTracks > 0 &&
		numTracks >= metaTracks && // no hidden tracks
		numTracks == streamableTracks {
		addURL(pageURL, seed.LinkType_FreeStreaming_Release_URL)
	}
	// Check if the page has a Creative Commons license.
	if lu, err := page.Query("div#license a.cc-icons").Attr("href"); err == nil {
		addURL(lu, seed.LinkType_License_Release_URL)
	}

	// If there's a back link to a label, prefill the search field and/or MBID.
	var labelName, labelMBID string
	if res := page.Query("a.back-to-label-link span.back-link-text"); res.Err == nil {
		if n := res.Node.LastChild; n != nil && n.Type == html.TextNode {
			labelName = strings.TrimSpace(n.Data)
		}
	}
	if val, err := page.Query("a.back-to-label-link").Attr("href"); err == nil {
		if labelURL, err := url.Parse(val); err == nil {
			labelURL.RawQuery = "" // clear "?from=btl"
			labelMBID = internal.GetLabelMBIDFromURL(shortCtx, db, labelURL.String(), labelName)
		}
	}
	// If we didn't find a label MBID yet, check if the base URL corresponds to a label.
	// Do this even if it already got matched to an artist, since sometimes the same Bandcamp
	// page gets used for an artist-owned label.
	if labelMBID == "" && baseURL != "" {
		labelMBID = internal.GetLabelMBIDFromURL(shortCtx, db, baseURL, labelName)
	}
	if labelName != "" || labelMBID != "" {
		rel.Labels = append(rel.Labels, seed.ReleaseLabel{
			Name: labelName,
			MBID: labelMBID,
		})
	}

	// If there aren't any media besides the digital download, seed the UPC if present.
	// (The userscript's justification for this is that "UPCs generally apply to physical
	// releases".)
	if len(album.Packages) == 0 && album.Current.UPC != "" {
		rel.Barcode = album.Current.UPC
	}

	// Fill unset fields where possible.
	rel.Autofill(shortCtx, !cfg.DisallowNetwork)

	// Add an informational edit containing the full-resolution cover art to make it easy
	// for the user to add it in a followup edit.
	// TODO: Is there any way to seed the image in the original edit?
	if iurl, err := page.Query("div#tralbumArt a").Attr("href"); err == nil {
		if strings.HasSuffix(iurl, "_10.jpg") {
			iurl = iurl[:len(iurl)-7] + "_0.jpg"
			var err error
			if img, err = seed.NewInfo("[cover image]", iurl); err != nil {
				return nil, nil, err
			}
		}
	}

	return rel, img, nil
}

// unmarshalAttr selects the element matched by query and JSON-unmarshals attr.
func unmarshalAttr(page *web.Page, query, attr string, dst interface{}) error {
	val, err := page.Query(query).Attr(attr)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dst)
}

var (
	// bandcampLaunch contains the Bandcamp launch date:
	// https://blog.bandcamp.com/2008/09/16/hello-cleveland/
	bandcampLaunch = time.Date(2008, 9, 16, 0, 0, 0, 0, time.UTC)
	// metaDescRegexp extracts the track count from a <meta property="og:description"> tag's content.
	metaDescRegexp = regexp.MustCompile(`^(\d+) track album$`)
)

// albumData corresponds to the data-tralbum JSON object embedded in Bandcamp album pages,
// which appears to be loaded into window.TralbumData.
// I admire Bandcamp's impartiality in the camelCase vs. snake_case conflict.
type albumData struct {
	Artist   string        `json:"artist"`
	Packages []interface{} `json:"packages"`
	HasAudio bool          `json:"hasAudio"`
	Current  struct {
		Title            string   `json:"title"`
		Type             string   `json:"type"`
		ReleaseDate      jsonDate `json:"release_date"`
		PublishDate      jsonDate `json:"publish_date"`
		UPC              string   `json:"upc"`
		DownloadPref     int      `json:"download_pref"`
		MinimumPrice     float64  `json:"minimum_price"`
		FreeDownloadPage string   `json:"freeDownloadPage"`
	} `json:"current"`
	TrackInfo []struct {
		Title    string            `json:"title"`
		Duration float64           `json:"duration"`
		File     map[string]string `json:"file"`
	} `json:"trackinfo"`
}

// jsonDate unmarshals a time provided as a JSON string like "07 Oct 2022 00:00:00 GMT".
type jsonDate time.Time

func (d *jsonDate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*d = jsonDate(time.Time{})
		return nil
	}
	t, err := time.Parse("02 Jan 2006 15:04:05 MST", s)
	*d = jsonDate(t)
	return err
}

func (d jsonDate) String() string { return time.Time(d).String() }

// embedData corresponds to the data-embed JSON object embedded in Bandcamp album pages,
// which appears to be loaded into window.EmbedData.
type embedData struct {
	AlbumEmbedData struct {
		Linkback string `json:"linkback"`
	} `json:"album_embed_data"`
}

var (
	// TODO: I'm just guessing here based on what I've seen.
	hostRegexp = regexp.MustCompile(`^[-a-z0-9]+\.bandcamp\.com$`)
	pathRegexp = regexp.MustCompile(`^/(?:album|track)/[-a-z0-9]+$`)
)

func getBaseURL(orig string) string {
	u, err := url.Parse(orig)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(u.Host, ".bandcamp.com") {
		return "https://" + u.Host + "/"
	}
	return ""
}

// extractTrackArtists attempts to extract one or more artist names from the beginning
// of the supplied track title.
func extractTrackArtists(orig string) (track string, artists []seed.ArtistCredit) {
	parts := strings.SplitN(orig, " - ", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return orig, nil
	}
	return parts[1], parseArtists(parts[0])
}

// parseArtists splits a string like "A, B & C" into individual artist credits.
func parseArtists(orig string) []seed.ArtistCredit {
	if orig == "" {
		return nil
	}

	// Split on join phrases and get the artist name from the part before each join phrase.
	ms := joinPhraseRegexp.FindAllStringIndex(orig, -1)
	if len(ms) == 0 {
		return []seed.ArtistCredit{{Name: orig}}
	}
	artists := make([]seed.ArtistCredit, len(ms)+1)
	for i, rng := range ms {
		start, end := rng[0], rng[1]
		artists[i].JoinPhrase = orig[start:end]

		var prev int
		if i > 0 {
			prev = ms[i-1][1]
		}
		if prev < start {
			artists[i].Name = orig[prev:start]
		}
	}

	// Add the artist after the final join phrase.
	if last := ms[len(ms)-1][1]; last < len(orig) {
		artists[len(artists)-1].Name = orig[last:]
	}

	// If any of the artist names were blank, just give up.
	for i := range artists {
		if artists[i].Name == "" {
			return []seed.ArtistCredit{{Name: orig}}
		}
	}

	return artists
}

// joinPhraseRegexp matches join phrases appearing in artist names.
var joinPhraseRegexp = regexp.MustCompile(`(?i)` + strings.Join([]string{
	` & `,
	`, `,
	` feat\. `,
	` ft\. `,
	// TODO: Add more? I think that these are freeform based on whatever the artist
	// enters when uploading their music. I've seen " x " used occasionally, but I'm
	// a bit worried about false positives.
}, "|"))

// CleanURL returns a cleaned version of a Bandcamp URL like
// "https://artist-name.bandcamp.com/album/album-name" or
// "https://artist-name.bandcamp.com/track/track-name".
// An error is returned if the URL doesn't match this format.
func (p *Provider) CleanURL(orig string) (string, error) {
	u, err := url.Parse(strings.ToLower(orig))
	if err != nil {
		return "", err
	}
	if !hostRegexp.MatchString(u.Host) {
		return "", errors.New(`host not "<name>.bandcamp.com"`)
	}
	if !pathRegexp.MatchString(u.Path) {
		return "", errors.New(`path not "/album/<name>" or "/track/<name>"`)
	}
	u.Scheme = "https"
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func (p *Provider) NeedsPage() bool    { return true }
func (p *Provider) ExampleURL() string { return "https://artist.bandcamp.com/album/…" }
