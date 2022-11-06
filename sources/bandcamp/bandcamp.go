// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package bandcamp fetches music information from Bandcamp.
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

	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/web"
	"golang.org/x/net/html"
)

// editNote is appended to automatically-generated edit notes.
const editNote = "\n\n(seeded using https://github.com/derat/yambs)"

// Fetch generates seeded edits from the Bandcamp page at url.
// This is heavily based on the bandcamp_importer.user.js userscript:
// https://github.com/murdos/musicbrainz-userscripts/blob/master/bandcamp_importer.user.js
func Fetch(ctx context.Context, url string) ([]seed.Edit, error) {
	page, err := web.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}
	rel, img, err := parsePage(page, url)
	if err != nil {
		return nil, err
	}
	edits := []seed.Edit{rel}
	if img != nil {
		edits = append(edits, img)
	}
	return edits, nil
}

// parsePage extracts release information from the supplied page.
// It's separate from Fetch to make testing easier.
func parsePage(page *web.Page, url string) (rel *seed.Release, img *seed.Info, err error) {
	// Upgrade the scheme for later usage.
	if strings.HasPrefix(url, "http://") {
		url = "https" + url[4:]
	}

	val, err := page.Query("script[data-tralbum]").Attr("data-tralbum")
	if err != nil {
		return nil, nil, err
	}
	var data albumData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, nil, err
	}
	// TODO: Support more types. The userscript maps "track" to a single release type,
	// unless "album_embed_data" is set, in which case it treats it as an individual
	// track on a parent album.
	if data.Current.Type != "album" {
		return nil, nil, fmt.Errorf("non-album type %q", data.Current.Type)
	}

	rel = &seed.Release{
		Title: data.Current.Title,
		// TODO: Add logic for detecting "various artists", maybe.
		// The userscript checks if all tracks have titles like "artist - tracktitle" with
		// non-numeric artists (which would instead be a track number) and tests the album
		// artist against '^various(?: artists)?$'.
		Artists:   []seed.ArtistCredit{{Name: data.Artist}},
		Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
		Status:    seed.ReleaseStatus_Official,
		Packaging: seed.ReleasePackaging_None,
		// The userscript appears to hardcode these too, but it might not be too hard
		// to at least detect the script.
		Language: "eng",
		Script:   "Latn",
		Mediums:  []seed.Medium{seed.Medium{Format: seed.MediumFormat_DigitalMedia}},
		EditNote: url + editNote,
	}

	// I'm guessing that the publish date is when the album was created in Bandcamp,
	// while the release date is when it was actually made available to users (but
	// can maybe also be set to some arbitrary date?). Follow the userscript's logic
	// of using the release date unless it precedes Bandcamp's launch.
	date := time.Time(data.Current.ReleaseDate)
	if date.IsZero() || date.Before(bandcampLaunch) {
		date = time.Time(data.Current.PublishDate)
	}
	if !date.IsZero() {
		rel.Events = []seed.ReleaseEvent{{
			Year:    date.Year(),
			Month:   int(date.Month()),
			Day:     date.Day(),
			Country: "XW",
		}}
	}

	// Add a single medium with all of the tracks.
	med := &rel.Mediums[0]
	var streamableTracks int
	for _, tr := range data.TrackInfo {
		// TODO: If we previously guessed that this release has various artists,
		// try to parse them from the title here. The userscript uses "^(.+) - (.+)$",
		// but the MB data also has an "artist" field -- is it used? Seems to be null
		// for single-artist albums.
		med.Tracks = append(med.Tracks, seed.Track{
			Title:  tr.Title,
			Length: time.Duration(float64(time.Second) * tr.Duration),
		})
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
	if pref := data.Current.DownloadPref; pref != 0 {
		if data.Current.FreeDownloadPage != "" ||
			pref == 1 ||
			(pref == 2 && data.Current.MinimumPrice == 0) {
			addURL(url, seed.LinkType_DownloadForFree_Release_URL)
		}
		if pref == 2 {
			addURL(url, seed.LinkType_PurchaseForDownload_Release_URL)
		}
	}
	if numTracks := len(med.Tracks); data.HasAudio && numTracks > 0 &&
		numTracks >= metaTracks && // no hidden tracks
		numTracks == streamableTracks {
		addURL(url, seed.LinkType_StreamingMusic_Release_URL)
	}
	// Check if the page has a Creative Commons license.
	if lurl, err := page.Query("div#license a.cc-icons").Attr("href"); err == nil {
		addURL(lurl, seed.LinkType_License_Release_URL)
	}

	// If there's a back link to a label, prefill the search field.
	// TODO: The userscript appears to also use the link URL to search for the label's MBID.
	if res := page.Query("a.back-to-label-link span.back-link-text"); res.Err == nil {
		if n := res.Node.LastChild; n != nil && n.Type == html.TextNode {
			rel.Labels = append(rel.Labels, seed.ReleaseLabel{
				Name: strings.TrimSpace(n.Data),
			})
		}
	}

	// If there aren't any media besides the digital download, seed the UPC if present.
	// (The userscript's justification for this is that "UPCs generally apply to physical
	// releases".)
	if len(data.Packages) == 0 && data.Current.UPC != "" {
		rel.Barcode = data.Current.UPC
	}

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

var (
	// bandcampLaunch contains the Bandcamp launch date:
	// https://blog.bandcamp.com/2008/09/16/hello-cleveland/
	bandcampLaunch = time.Date(2008, 9, 16, 0, 0, 0, 0, time.UTC)
	// metaDescRegexp extracts the track count from a <meta property="og:description"> tag's content.
	metaDescRegexp = regexp.MustCompile(`^(\d+) track album$`)
)

// albumData corresponds to the data-tralbum JSON object embedded in Bandcamp album pages.
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
	t, err := time.Parse("02 Jan 2006 15:04:05 MST", s)
	*d = jsonDate(t)
	return err
}

func (d jsonDate) String() string { return time.Time(d).String() }

var (
	// TODO: I'm just guessing here based on what I've seen.
	hostRegexp      = regexp.MustCompile(`^[-a-z0-9]+\.bandcamp\.com$`)
	albumPathRegexp = regexp.MustCompile(`^/album/[-a-z0-9]+$`)
)

// CleanURL returns a cleaned version of a Bandcamp URL like
// "https://artist-name.bandcamp.com/album/album-name".
// An error is returned if the URL doesn't match this format.
func CleanURL(orig string) (string, error) {
	u, err := url.Parse(strings.ToLower(orig))
	if err != nil {
		return "", err
	}
	if !hostRegexp.MatchString(u.Host) {
		return "", errors.New(`host not "<name>.bandcamp.com"`)
	}
	if !albumPathRegexp.MatchString(u.Path) {
		return "", errors.New(`path not "/album/<name>"`)
	}
	u.Scheme = "https"
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
