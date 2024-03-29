// Copyright 2023 Daniel Erat.
// All rights reserved.

// Package tidal uses Tidal's API to seed edits.
package tidal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online/internal"
	"github.com/derat/yambs/web"
)

const (
	// defaultToken is used to access the Tidal API.
	// This particular token looks like it's been working since May 2020, if not earlier:
	// https://github.com/spencercharest/tidal-api/issues/12
	defaultToken = "gsFXkJqGrUNoYMQPZe4k3WKwijnrp8iGSwn3bApe"
	// defaultCountry is the default country code passed to the Tidal API.
	defaultCountry = "US"
	// finishTime is reserved to finish creating edits after querying the MusicBrainz API.
	finishTime = 3 * time.Second
	// maxContriesForAnnotation is the maximum number of countries to include in annotations
	// for albums that aren't available everywhere.
	maxCountriesForAnnotation = 10
)

// https://www.telegraph.co.uk/technology/news/11192375/Tidal-launches-lossless-music-streaming-in-UK-and-US.html
var tidalLaunch = time.Date(2014, 10, 28, 0, 0, 0, 0, time.UTC)

// Provider implements internal.Provider for Tidal.
type Provider struct{}

// Release generates a seeded release edit for the supplied Tidal album URL.
// Tidal provides a JSON API, so the page parameter is not used.
func (p *Provider) Release(ctx context.Context, page *web.Page, pageURL string,
	db *mbdb.DB, cfg *internal.Config) (rel *seed.Release, img *seed.Info, err error) {
	if cfg.DisallowNetwork {
		return nil, nil, errors.New("network is disallowed")
	}
	api := newRealAPICaller(defaultToken)
	return getRelease(ctx, pageURL, api, db, cfg, time.Now())
}

// getRelease is called by Release.
// This helper function exists so that unit tests can inject fake apiCallers.
func getRelease(ctx context.Context, pageURL string, api apiCaller, db *mbdb.DB, cfg *internal.Config,
	now time.Time) (rel *seed.Release, img *seed.Info, err error) {
	albumURL, err := cleanURL(pageURL)
	if err != nil {
		return nil, nil, err
	}
	urlParts := strings.Split(albumURL, "/")
	albumID, err := strconv.Atoi(urlParts[len(urlParts)-1])
	if err != nil {
		return nil, nil, fmt.Errorf("album ID: %v", err)
	}

	country := cfg.CountryCode
	if country == "" {
		country = defaultCountry
	}
	album, credits, countryTracklists, err := fetchAllData(ctx, api, albumID, country)
	if err != nil {
		return nil, nil, err
	}

	if album.NumberOfTracks <= 0 {
		return nil, nil, fmt.Errorf("API claimed album has %d tracks", album.NumberOfTracks)
	}

	rel = &seed.Release{
		Title:     album.Title,
		Status:    seed.ReleaseStatus_Official,
		Packaging: seed.ReleasePackaging_None,
		Barcode:   album.UPC,
	}

	switch album.Type {
	case "ALBUM":
		rel.Types = append(rel.Types, seed.ReleaseGroupType_Album)
	case "EP":
		rel.Types = append(rel.Types, seed.ReleaseGroupType_EP)
	case "SINGLE":
		rel.Types = append(rel.Types, seed.ReleaseGroupType_Single)
	}

	if date := time.Time(album.ReleaseDate); !date.Before(tidalLaunch) {
		rel.Events = []seed.ReleaseEvent{{Date: seed.DateFromTime(date)}}
	}

	var annotations []string

	if cp := strings.TrimSpace(album.Copyright); cp != "" {
		// The copyright field is a mess; I've seen all of the following:
		//  (P) 1992 Sony Music Entertainment
		//  2016 M83 Recording Inc. under exclusive license to Mute for North America
		//  © 2018 AIC Entertainment, LLC under exclusive license to BMG Rights Management (US) LLC
		//  © 2008 Atlantic Recording Corporation for the United States and WEA International Inc.
		//    for the world outside of the United States.
		// Prepend the word "Copyright" if there's nothing making it clear that this is a copyright.
		if !hasCopyright(cp) {
			cp = "Copyright " + cp
		}
		annotations = append(annotations, cp)
	}

	for _, cred := range credits {
		if cred.Type == "Record Label" {
			for _, cont := range cred.Contributors {
				rel.Labels = append(rel.Labels, seed.ReleaseLabel{Name: cont.Name})
			}
			break
		}
	}

	// Check that we have the full tracklist before going to the trouble of looking up artists.
	var tracklist *tracklistData
	if country != AllCountriesCode {
		if tracklist = countryTracklists[country]; tracklist == nil {
			return nil, nil, errors.New("didn't get tracklist") // shouldn't happen
		} else if len(tracklist.Items) != album.NumberOfTracks {
			return nil, nil, fmt.Errorf("got %d track(s) instead of %d (is album only partially-available in %q?)",
				len(tracklist.Items), album.NumberOfTracks, country)
		}
	} else {
		var fullCountries []string
		for c, tl := range countryTracklists {
			if len(tl.Items) == album.NumberOfTracks {
				if tracklist == nil {
					tracklist = tl
				}
				fullCountries = append(fullCountries, c)
			}
		}
		if tracklist == nil {
			return nil, nil, fmt.Errorf("no country has full tracklist with %d track(s)", album.NumberOfTracks)
		}
		if len(fullCountries) <= maxCountriesForAnnotation {
			annotations = append(annotations, makeCountriesAnnotation(fullCountries, now))
		}
	}

	if len(annotations) > 0 {
		rel.Annotation = strings.Join(annotations, "\n\n")
	}

	// Use a shortened context for querying MusicBrainz for artist MBIDs so we'll have a bit of time
	// left to finish creating the edit even if we need to look up a bunch of different artists:
	// https://github.com/derat/yambs/issues/19
	shortCtx, shortCancel := mbdb.ShortenContext(ctx, finishTime)
	defer shortCancel()

	rel.Artists = makeArtistCredits(shortCtx, album.Artists, db)

	var vol int // last-seen volume number
	for _, tr := range tracklist.Items {
		// Start a new disk when needed.
		if len(rel.Mediums) == 0 || tr.VolumeNumber != vol {
			rel.Mediums = append(rel.Mediums, seed.Medium{Format: seed.MediumFormat_DigitalMedia})
			vol = tr.VolumeNumber
		}
		track := seed.Track{
			// TODO: tr.Title sometimes contains a redundant " (feat. Some Artist)" suffix that's
			// also included in tr.Artists. Other times, the featured artist is missing from
			// tr.Artists but still included in tr.Title. Consider automatically removing the suffix
			// from the title when it's also in the artist list.
			Title:  tr.Title,
			Length: time.Duration(tr.Duration) * time.Second,
			// TODO: Find some way to seed ISRCs.
		}
		// Don't assign artist credits to the track if they'd be identical to the album credits.
		if !reflect.DeepEqual(tr.Artists, album.Artists) {
			track.Artists = makeArtistCredits(shortCtx, tr.Artists, db)
		}
		med := &rel.Mediums[len(rel.Mediums)-1]
		med.Tracks = append(med.Tracks, track)
	}

	rel.URLs = append(rel.URLs, seed.URL{
		URL:      albumURL,
		LinkType: seed.LinkType_Streaming_Release_URL,
	})

	// Autofill the language and script.
	rel.Autofill(ctx, !cfg.DisallowNetwork)

	if album.Cover != "" {
		// https://rateyourmusic.com/discussion/rate-your-music/tip-scraping-cover-art-from-tidal-for-streaming-only-releases/
		iurl := "https://resources.tidal.com/images/" + strings.ReplaceAll(album.Cover, "-", "/") + "/origin.jpg"
		if img, err = seed.NewInfo("[cover image]", iurl); err != nil {
			return nil, nil, err
		}
	}

	return rel, img, nil
}

// makeArtistCredits constructs a slice of seed.ArtistCredit objects
// based on the supplied artist list from the API.
func makeArtistCredits(ctx context.Context, artists []artistData, db *mbdb.DB) []seed.ArtistCredit {
	credits := make([]seed.ArtistCredit, len(artists))
	for i, a := range artists {
		credits[i].Name = a.Name

		// Try to look up the artist's MBID based on their canonical URL.
		if a.ID != 0 {
			aurl := fmt.Sprintf("https://tidal.com/artist/%d", a.ID)
			credits[i].MBID = internal.GetArtistMBIDFromURL(ctx, db, aurl, a.Name)
		}

		if i > 0 {
			// TODO: Handle other types if they exist.
			switch a.Type {
			case "FEATURED":
				credits[i-1].JoinPhrase = " feat. "
			default: // "MAIN"
				credits[i-1].JoinPhrase = " & "
				if pi := i - 2; pi >= 0 && credits[pi].JoinPhrase == " & " {
					credits[pi].JoinPhrase = ", "
				}
			}
		}
	}
	return credits
}

// CleanURL returns a cleaned version of a Tidal album URL:
//  https://tidal.com/album/1234 (MB's canonical form, redirects to /browse/album/1234)
//  https://tidal.com/browse/album/1234 (Tidal's canonical form)
//  https://listen.tidal.com/album/1234 (streaming page)
// An error is returned if the URL doesn't match this format.
func (p *Provider) CleanURL(orig string) (string, error) { return cleanURL(orig) }

func cleanURL(orig string) (string, error) {
	u, err := url.Parse(strings.ToLower(orig))
	if err != nil {
		return "", err
	}
	if u.Host != "tidal.com" && u.Host != "listen.tidal.com" {
		return "", errors.New(`host not "tidal.com" or "listen.tidal.com"`)
	}
	if ms := pathRegexp.FindStringSubmatch(u.Path); ms == nil {
		return "", errors.New(`path not "/album/<id>"`)
	} else {
		u.Path = ms[1]
	}
	u.Scheme = "https"
	u.Host = "tidal.com"
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// pathRegexp matches a Tidal album URL path.
// The first match group contains the canonical portion.
var pathRegexp = regexp.MustCompile(`^(?:/browse)?(/album/\d+)$`)

func (p *Provider) NeedsPage() bool    { return false }
func (p *Provider) ExampleURL() string { return "https://tidal.com/album/…" }

// hasCopyright returns true if s contains a copyright or phonogram symbol
// or the word "copyright".
func hasCopyright(s string) bool {
	ls := strings.ToLower(s)
	for _, sym := range []string{
		"©",
		"℗",
		"(c)",
		"(p)",
		"copyright",
	} {
		if strings.Contains(ls, sym) {
			return true
		}
	}
	return false
}

// makeCountriesAnnotation returns a string for seed.Release's Annotation field
// containing the supplied list of countries where an album is available.
func makeCountriesAnnotation(countries []string, now time.Time) string {
	vals := make([]string, len(countries))
	for i, c := range countries {
		vals[i] = fmt.Sprintf("    * %s (%s)", allCountries[c], c)
	}
	sort.Strings(vals)
	date := now.UTC().Format("2006-01-02")
	return "Regions with all tracks on Tidal (as of " + date + " UTC):\n" + strings.Join(vals, "\n")
}

// AllCountriesCode is a value for online.Config's CountryCode field indicating that all countries
// should be queried.
const AllCountriesCode = "XW"

// allCountries maps from ISO 3166 codes to names for all countries/regions where Tidal is
// available per https://support.tidal.com/hc/en-us/articles/202453191-TIDAL-Where-We-re-Available.
// Turkey also seems to be supported by the API, even though it's not listed as of 2022-02-06.
var allCountries = map[string]string{
	"AL": "Albania", // unsupported?
	"AD": "Andorra",
	"AR": "Argentina",
	"AU": "Australia",
	"AT": "Austria",
	"BE": "Belgium",
	"BA": "Bosnia and Herzegovina", // unsupported?
	"BR": "Brazil",
	"BG": "Bulgaria", // unsupported?
	"CA": "Canada",
	"CL": "Chile",
	"CO": "Colombia",
	"HR": "Croatia", // unsupported?
	"CY": "Cyprus",
	"CZ": "Czech Republic",
	"DK": "Denmark",
	"DO": "Dominican Republic",
	"EE": "Estonia",
	"FI": "Finland",
	"FR": "France",
	"DE": "Germany",
	"GR": "Greece",
	"HK": "Hong Kong",
	"HU": "Hungary",
	"IS": "Iceland",
	"IE": "Ireland",
	"IL": "Israel",
	"IT": "Italy",
	"JM": "Jamaica",
	"LV": "Latvia",
	"LI": "Liechtenstein",
	"LT": "Lithuania",
	"LU": "Luxemburg",
	"MY": "Malaysia",
	"MT": "Malta",
	"MX": "Mexico",
	"MC": "Monaco",
	"ME": "Montenegro", // unsupported?
	"NL": "Netherlands",
	"NZ": "New Zealand",
	"NG": "Nigeria",
	"MK": "North Macedonia", // unsupported?
	"NO": "Norway",
	"PE": "Peru",
	"PL": "Poland",
	"PT": "Portugal",
	"PR": "Puerto Rico",
	"RO": "Romania",
	"RS": "Serbia", // unsupported?
	"SG": "Singapore",
	"SK": "Slovakia",
	"SI": "Slovenia",
	"ZA": "South Africa",
	"ES": "Spain",
	"SE": "Sweden",
	"CH": "Switzerland",
	"TH": "Thailand",
	"TR": "Turkey",
	"UG": "Uganda",
	"AE": "United Arab Emirates", // unsupported?
	"GB": "United Kingdom",
	"US": "United States of America",
}
