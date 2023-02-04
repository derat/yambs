// Copyright 2023 Daniel Erat.
// All rights reserved.

// Package tidal uses Tidal's API to seed edits.
package tidal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/derat/yambs/db"
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
)

// https://www.telegraph.co.uk/technology/news/11192375/Tidal-launches-lossless-music-streaming-in-UK-and-US.html
var tidalLaunch = time.Date(2014, 10, 28, 0, 0, 0, 0, time.UTC)

// Provider implements internal.Provider for Tidal.
type Provider struct{}

// Release generates a seeded release edit for the supplied Tidal album URL.
// Tidal provides a JSON API, so the page parameter is not used.
func (p *Provider) Release(ctx context.Context, page *web.Page, pageURL string,
	db *db.DB, cfg *internal.Config) (rel *seed.Release, img *seed.Info, err error) {
	if cfg.DisallowNetwork {
		return nil, nil, errors.New("network is disallowed")
	}
	api := newRealAPICaller(defaultToken)
	return getRelease(ctx, pageURL, api, db, cfg)
}

// getRelease is called by Release.
// This helper function exists so that unit tests can inject fake apiCallers.
func getRelease(ctx context.Context, pageURL string, api apiCaller, db *db.DB, cfg *internal.Config) (
	rel *seed.Release, img *seed.Info, err error) {
	albumURL, err := cleanURL(pageURL)
	if err != nil {
		return nil, nil, err
	}
	urlParts := strings.Split(albumURL, "/")
	albumID, err := strconv.Atoi(urlParts[len(urlParts)-1])
	if err != nil {
		return nil, nil, fmt.Errorf("album ID: %v", err)
	}

	country := defaultCountry
	if cfg.CountryCode != "" {
		country = strings.ToUpper(cfg.CountryCode)
	}
	if !countryCodeRegexp.MatchString(country) {
		return nil, nil, fmt.Errorf("invalid country code %q (want two capital letters)", country)
	}

	var album albumData
	if r, err := api.call(ctx, fmt.Sprintf("/v1/albums/%d?countryCode=%s", albumID, country)); err != nil {
		return nil, nil, fmt.Errorf("fetching album: %v", err)
	} else {
		defer r.Close()
		if err := json.NewDecoder(r).Decode(&album); err != nil {
			return nil, nil, fmt.Errorf("decoding album: %v", err)
		}
	}
	if album.NumberOfTracks <= 0 {
		return nil, nil, fmt.Errorf("API claimed album has %d track(s)", album.NumberOfTracks)
	}

	missingArtistIDs := make(map[int]bool) // cache of artists not in MB
	rel = &seed.Release{
		Title:     album.Title,
		Artists:   makeArtistCredits(ctx, album.Artists, db, missingArtistIDs),
		Status:    seed.ReleaseStatus_Official,
		Packaging: seed.ReleasePackaging_None,
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
		// TODO: Set the release location(s)?
		rel.Events = []seed.ReleaseEvent{{Date: seed.DateFromTime(date)}}
	}

	var tracklist tracklistData
	if r, err := api.call(ctx, fmt.Sprintf("/v1/albums/%d/tracks?countryCode=%s", albumID, country)); err != nil {
		return nil, nil, fmt.Errorf("fetching tracklist: %v", err)
	} else {
		defer r.Close()
		if err := json.NewDecoder(r).Decode(&tracklist); err != nil {
			return nil, nil, fmt.Errorf("decoding tracklist: %v", err)
		}
	}
	if len(tracklist.Items) != album.NumberOfTracks {
		return nil, nil, fmt.Errorf("API claimed %v track(s) but only returned %v (album unavailable in %v?)",
			album.NumberOfTracks, len(tracklist.Items), country)
	}

	sort.Slice(tracklist.Items, func(i, j int) bool {
		ti, tj := tracklist.Items[i], tracklist.Items[j]
		return ti.VolumeNumber < tj.VolumeNumber ||
			(ti.VolumeNumber == tj.VolumeNumber && ti.TrackNumber < tj.TrackNumber)
	})

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
			track.Artists = makeArtistCredits(ctx, tr.Artists, db, missingArtistIDs)
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

var countryCodeRegexp = regexp.MustCompile(`^[A-Z][A-Z]$`)

type albumData struct {
	ID             int          `json:"id"`
	Title          string       `json:"title"` // album title
	AllowStreaming bool         `json:"allowStreaming"`
	NumberOfTracks int          `json:"numberOfTracks"`
	ReleaseDate    jsonDate     `json:"releaseDate"` // e.g. "2016-06-24"
	Type           string       `json:"type"`        // "ALBUM", "EP", "SINGLE"
	Cover          string       `json:"cover"`       // UUID
	Artist         artistData   `json:"artist"`
	Artists        []artistData `json:"artists"`
}

type artistData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // e.g. "MAIN", "FEATURED"
}

type tracklistData struct {
	Items []trackData `json:"items"`
	// TODO: Is the tracklist automatically paginated in some cases?
	// There are also "limit", "offset", and "totalNumberOfItems" fields.
}

type trackData struct {
	ID             int          `json:"id"`
	Title          string       `json:"title"`
	Duration       float32      `json:"duration"` // e.g. 364
	AllowStreaming bool         `json:"allowStreaming"`
	TrackNumber    int          `json:"trackNumber"`
	VolumeNumber   int          `json:"volumeNumber"`
	URL            string       `json:"url"` // e.g. "http://www.tidal.com/track/1234"
	ISRC           string       `json:"isrc"`
	Artist         artistData   `json:"artist"`  // contains only main
	Artists        []artistData `json:"artists"` // contains both main and featured
}

// jsonDate unmarshals a time provided as a JSON string like "2020-05-21".
type jsonDate time.Time

func (d *jsonDate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", s)
	*d = jsonDate(t)
	return err
}

func (d jsonDate) String() string { return time.Time(d).String() }

// makeArtistCredits constructs a slice of seed.ArtistCredit objects
// based on the supplied artist list from the API.
// missing is a cache of Tidal artist IDs that are known to not be in MusicBrainz;
// the same map should be passed to successive calls to this function..
func makeArtistCredits(ctx context.Context, artists []artistData,
	db *db.DB, missing map[int]bool) []seed.ArtistCredit {
	credits := make([]seed.ArtistCredit, len(artists))
	for i, a := range artists {
		credits[i].Name = a.Name

		// Try to look up the artist's MBID based on their URL. MusicBrainz appears to normalize
		// Tidal URLs to tidal.com now, but I still see a lot of "Stream at Tidal" relationships
		// with stream.tidal.com URLs. Look for both, I guess:
		// https://github.com/derat/yambs/issues/20
		if a.ID != 0 && !missing[a.ID] {
			var mbid string
			for _, aurl := range []string{
				fmt.Sprintf("https://tidal.com/artist/%d", a.ID),
				fmt.Sprintf("https://stream.tidal.com/artist/%d", a.ID),
			} {
				var err error
				if mbid, err = db.GetArtistMBIDFromURL(ctx, aurl); err != nil {
					log.Printf("Failed getting MBID for %v: %v", aurl, err)
				} else if mbid != "" {
					break
				}
			}
			if mbid != "" {
				credits[i].MBID = mbid
			} else {
				missing[a.ID] = true // don't try again
			}
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

// apiCaller calls the Tidal API. This interface exists so fake instances can be injected by tests.
type apiCaller interface {
	// call makes a GET request to the Tidal API using the specified path (i.e. "/v1/...").
	call(ctx context.Context, path string) (io.ReadCloser, error)
}

// realAPICaller is an apiCaller implementation that calls the real Tidal API.
type realAPICaller struct {
	client *http.Client
	token  string
}

func newRealAPICaller(token string) *realAPICaller {
	return &realAPICaller{
		client: &http.Client{
			Transport: &http.Transport{
				// TODO: Find some way to avoid needing this.
				// I (sometimes?) get "net/http: TLS handshake timeout" when I use http.DefaultClient.
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		token: token,
	}
}

func (api *realAPICaller) call(ctx context.Context, path string) (io.ReadCloser, error) {
	url := "https://api.tidal.com" + path
	log.Print("Fetching ", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Tidal-Token", api.token)
	res, err := api.client.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode != 200 {
		res.Body.Close()
		return nil, fmt.Errorf("status %v: %v", res.StatusCode, res.Status)
	}
	return res.Body, nil
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
