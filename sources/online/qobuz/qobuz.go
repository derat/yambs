// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package qobuz extracts information from Qobuz pages.
package qobuz

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
)

// Provider implements internal.Provider for Qobuz.
type Provider struct{}

// pathRegexp matches a path like "/album/hyttetur-2-svartepetter/e3qy2e01fbs9a" or
// "/us-en/album/in-rainbows-radiohead/0634904032432". The first match group is the
// (optional) path component containing the country and language codes. The second
// match group contains the rest of the path.
var pathRegexp = regexp.MustCompile(`^(/[a-z]{2}-[a-z]{2})?(/album/[^/]+/[^/]+)/?$`)

// CleanURL returns a cleaned version of a Qobuz URL like
// "https://www.qobuz.com/gb-en/album/album-name/album-id".
func (p *Provider) CleanURL(orig string) (string, error) {
	// Preserve the original locale. If we remove it, Qobuz will redirect to the artist's
	// discography page if the album isn't available in our country:
	// https://github.com/derat/yambs/issues/8
	return cleanURL(orig, false /* removeLocal */)
}

// cleanURL is a helper function called by CleanURL.
// If removeLocale is true, the e.g. "/gb-en" path component is dropped.
func cleanURL(orig string, removeLocale bool) (string, error) {
	u, err := url.Parse(strings.ToLower(orig))
	if err != nil {
		return "", err
	}
	if u.Host != "www.qobuz.com" {
		return "", errors.New(`host not "www.qobuz.com"`)
	}
	if ms := pathRegexp.FindStringSubmatch(u.Path); ms == nil {
		return "", errors.New(`path not "/<locale>/album/<name>/<id>"`)
	} else if removeLocale {
		u.Path = ms[2]
	} else {
		u.Path = ms[1] + ms[2]
	}
	u.Scheme = "https"
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func (p *Provider) NeedsPage() bool    { return true }
func (p *Provider) ExampleURL() string { return "https://www.qobuz.com/us-en/album/…" }

// Release extracts release information from the supplied Qobuz page.
func (p *Provider) Release(ctx context.Context, page *web.Page, pageURL string,
	db *mbdb.DB, cfg *internal.Config) (rel *seed.Release, img *seed.Info, err error) {
	// Get the page title so it can be included in errors.
	pageTitle, _ := page.Query("title").Text(true)

	rel = &seed.Release{
		Status:    seed.ReleaseStatus_Official,
		Packaging: seed.ReleasePackaging_None,
		Mediums:   []seed.Medium{{Format: seed.MediumFormat_DigitalMedia}},
	}

	// Extract the album title from the page since the version in the structured data seems to omit
	// extra title info. For example, the structured data in
	// https://www.qobuz.com/us-en/album/waynes-world-various-artists/0093624963714 contains
	// "Wayne's World" rather than the "Wayne's World (Music From The Motion Picture)" that appears
	// in the actual page.
	if rel.Title, err = page.Query(".album-meta__title").Text(true); err != nil {
		return nil, nil, fmt.Errorf("title (%q): %v", pageTitle, err)
	}

	// The HTML is a mess (e.g. the date format differs depending on the locale),
	// so get what we can from the structured data.
	var data structData
	if js, err := page.Query(`script[type="application/ld+json"]`).Text(true); err != nil {
		return nil, nil, fmt.Errorf("structured data (%q): %v", pageTitle, err)
	} else if err := json.Unmarshal([]byte(js), &data); err != nil {
		return nil, nil, fmt.Errorf("structured data: %v", err)
	} else if data.Context != "https://schema.org/" {
		return nil, nil, fmt.Errorf("structured data has unexpected context %q", data.Context)
	} else if data.Brand.Name == "" {
		return nil, nil, errors.New("structured data is missing artist")
	}
	rel.Artists = []seed.ArtistCredit{{Name: data.Brand.Name}}

	// Use the release date if it's plausible (i.e. not before Qobuz's launch).
	if t, err := time.Parse(`2006-01-02`, data.ReleaseDate); err == nil && !t.Before(qobuzLaunch) {
		rel.Events = []seed.ReleaseEvent{{Date: seed.DateFromTime(t)}}
	}

	// Add an informational edit containing the cover image URL.
	for _, src := range data.Image {
		// Change e.g. "_600.jpg" to "_max.jpg" to get the highest resolution available
		// (per https://wiki.musicbrainz.org/User:Nikki/CAA),
		if ms := imgRegexp.FindStringSubmatch(src); ms != nil {
			if img, err = seed.NewInfo("[cover image]", ms[1]+"_max.jpg"); err != nil {
				return nil, nil, err
			}
			break
		}
	}

	// Qobuz includes a list with extra data like the release date, label, "main artist", and genre.
	if extras := page.QueryAll(".album-meta__item"); extras.Err != nil {
		return nil, nil, fmt.Errorf("extras: %v", extras.Err)
	} else {
		for _, node := range extras.Nodes {
			// We already got the release date (in a stable format) from the structured data,
			// but get the label name from a line like "Released on 1/1/96 by Telarc".
			if dateRegexp.MatchString(web.GetText(node, true)) {
				if label, err := web.QueryNode(node, ".album-meta__link").Text(true); err == nil && label != "" {
					rel.Labels = []seed.ReleaseLabel{{Name: label}}
				}
			}
			// TODO: Try to also handle the artist link? It looks like this:
			//
			//  <li class="album-meta__item">Main artist:
			//    <a class="album-meta__link" href="/us-en/interpreter/dave-brubeck/4906"
			//      title="Dave Brubeck">Dave Brubeck</a>
			//  </li>
			//
			// I looked at a few popular artists and didn't see links to their Qobuz pages in MB,
			// so it may not be worth the effort. MusicBrainz doesn't seem to have Qobuz label
			// URLs like "/us-en/label/telarc-3/download-streaming-albums/247896" either.
		}
	}

	// Add tracks. Use span:first-child for titles to avoid picking up text from additional
	// spans, e.g. "Explicit".
	if tracks, err := page.QueryAll("#playerTracks .track__item--name span:first-child").Text(true); err != nil {
		return nil, nil, fmt.Errorf("track titles: %v", err)
	} else if artists, err := page.QueryAll("#playerTracks .track__item--artist span:first-child").Text(true); err != nil {
		return nil, nil, fmt.Errorf("track titles: %v", err)
	} else if durs, err := page.QueryAll("#playerTracks .track__item--duration").Text(true); err != nil {
		return nil, nil, fmt.Errorf("track durations: %v", err)
	} else if len(tracks) == 0 {
		return nil, nil, errors.New("didn't find track titles")
	} else if len(artists) > 0 && len(tracks) != len(artists) {
		return nil, nil, fmt.Errorf("found %d track titles(s) but %d artist(s)", len(tracks), len(artists))
	} else if len(tracks) != len(durs) {
		return nil, nil, fmt.Errorf("found %d track titles(s) but %d duration(s)", len(tracks), len(durs))
	} else {
		for i, title := range tracks {
			dur, err := parseDuration(durs[i])
			if err != nil {
				return nil, nil, fmt.Errorf("track %d duration %q: %v", i, durs[i], err)
			}
			tr := seed.Track{Title: title, Length: dur}
			if len(artists) > 0 {
				tr.Artists = []seed.ArtistCredit{{Name: artists[i]}}
			}
			rel.Mediums[0].Tracks = append(rel.Mediums[0].Tracks, tr)
		}
	}

	// Add URL relationships.
	if page.Query(".album-addtocart__add").Err == nil {
		// Remove the locale from the URL. The downside of this is that the locale-less URL may
		// just redirect to the discography for users in countries where the album isn't available,
		// but that still seems better than submitting whatever the editor's locale happens to be.
		cleaned, err := cleanURL(pageURL, true /* removeLocale */)
		if err != nil {
			return nil, nil, err
		}
		rel.URLs = append(rel.URLs, seed.URL{
			URL:      cleaned,
			LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
		})
	}
	if data.SKU != "" {
		rel.URLs = append(rel.URLs, seed.URL{
			URL:      "https://open.qobuz.com/album/" + data.SKU,
			LinkType: seed.LinkType_Streaming_Release_URL,
		})
	}

	// Fill unset fields where possible.
	rel.Autofill(ctx, !cfg.DisallowNetwork)

	return rel, img, nil
}

// structData represents JSON structured data within a
// <script type="application/ld+json"> element.
type structData struct {
	Context     string   `json:"@context"`
	Name        string   `json:"name"`
	Image       []string `json:"image"`
	SKU         string   `json:"sku"`
	ReleaseDate string   `json:"releaseDate"`
	Brand       struct {
		Name string `json:"name"`
	} `json:"brand"`
}

var (
	// qobuzLaunch contains Qobuz's launch date per https://en.wikipedia.org/wiki/Qobuz.
	qobuzLaunch = time.Date(2007, 9, 18, 0, 0, 0, 0, time.UTC)
	// dateRegexp matches a release date in a .album-meta__item list item.
	// The actual format depends on the locale.
	dateRegexp = regexp.MustCompile(` \d+/\d+/\d+ `)
	// imgRegexp matches a cover image URL.
	imgRegexp = regexp.MustCompile(`^(https://static\.qobuz\.com/images/covers/.+)_\d+\.jpg$`)
)

// parseDuration parses a duration in the form "HH:MM:SS".
func parseDuration(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, errors.New("not HH:MM:SS")
	}
	nums := make([]int, 3)
	for i, p := range parts {
		var err error
		if nums[i], err = strconv.Atoi(p); err != nil {
			return 0, err
		}
	}
	return time.Duration(nums[0])*time.Hour +
		time.Duration(nums[1])*time.Minute +
		time.Duration(nums[2])*time.Second, nil
}
