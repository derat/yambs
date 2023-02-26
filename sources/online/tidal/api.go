// Copyright 2023 Daniel Erat.
// All rights reserved.

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
	"sort"
	"time"
)

const (
	// apiTimeout is the maximum time for a request to the (unreliable) Tidal API.
	apiTimeout = 5 * time.Second
	// apiRetries contains the number of times to retry a failed (non-404) API call.
	apiRetries = 1
)

// apiCaller calls the Tidal API. This interface exists so fake instances can be injected by tests.
type apiCaller interface {
	// call makes a GET request to the Tidal API using the specified path (i.e. "/v1/...").
	call(ctx context.Context, path string) ([]byte, error)
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

var notFoundErr = errors.New("not found")

func (api *realAPICaller) call(ctx context.Context, path string) ([]byte, error) {
	url := "https://api.tidal.com" + path

	var tries int
	for {
		b, err, fatal := func() ([]byte, error, bool) {
			ctx, cancel := context.WithTimeout(ctx, apiTimeout)
			defer cancel()

			log.Print("Fetching ", url)
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, err, false
			}
			req.Header.Set("X-Tidal-Token", api.token)
			res, err := api.client.Do(req)
			if err != nil {
				return nil, err, false
			}
			defer res.Body.Close()

			switch res.StatusCode {
			case http.StatusOK:
				b, err := io.ReadAll(res.Body)
				return b, err, false
			case http.StatusNotFound:
				return nil, notFoundErr, true
			default:
				return nil, fmt.Errorf("status %v: %v", res.StatusCode, res.Status), false
			}
		}()
		tries++

		if err == nil {
			return b, nil
		} else if fatal || tries >= 1+apiRetries || ctx.Err() != nil {
			return nil, err // can't retry
		}
		log.Printf("Retryable error for %v: %v", url, err)
	}
}

// fetchAlbum fetches information about the specified album using api.
func fetchAlbum(ctx context.Context, api apiCaller, albumID int, country string) (*albumData, error) {
	var album albumData
	if b, err := api.call(ctx, fmt.Sprintf("/v1/albums/%d?countryCode=%s", albumID, country)); err != nil {
		return nil, err
	} else if err := json.Unmarshal(b, &album); err != nil {
		return nil, err
	}
	return &album, nil
}

// fetchCredits fetches information about the specified album's credits using api.
func fetchCredits(ctx context.Context, api apiCaller, albumID int, country string) (creditsData, error) {
	var credits creditsData
	if b, err := api.call(ctx, fmt.Sprintf("/v1/albums/%d/credits?countryCode=%s", albumID, country)); err != nil {
		return nil, err
	} else if err := json.Unmarshal(b, &credits); err != nil {
		return nil, err
	}
	return credits, nil
}

// fetchTracklist fetches information about the specified album's tracklist using api.
func fetchTracklist(ctx context.Context, api apiCaller, albumID int, country string) (*tracklistData, error) {
	var tracklist tracklistData
	if b, err := api.call(ctx, fmt.Sprintf("/v1/albums/%d/tracks?countryCode=%s", albumID, country)); err != nil {
		return nil, err
	} else if err := json.Unmarshal(b, &tracklist); err != nil {
		return nil, err
	}
	sort.Slice(tracklist.Items, func(i, j int) bool {
		ti, tj := tracklist.Items[i], tracklist.Items[j]
		return ti.VolumeNumber < tj.VolumeNumber ||
			(ti.VolumeNumber == tj.VolumeNumber && ti.TrackNumber < tj.TrackNumber)
	})
	return &tracklist, nil
}

// fetchAllTracklists calls fetchTracklist in parallel for all supported countries.
// The returned map is keyed by ISO 3166 country code.
// 404 errors are mapped to empty tracklists; any other error is returned alongside a nil map.
func fetchAllTracklists(ctx context.Context, api apiCaller, albumID int) (map[string]*tracklistData, error) {
	type result struct {
		country string
		td      *tracklistData
		err     error
	}
	ch := make(chan result, len(allCountries))
	for country := range allCountries {
		go func(country string) {
			td, err := fetchTracklist(ctx, api, albumID, country)
			ch <- result{country, td, err}
		}(country)
	}

	m := make(map[string]*tracklistData, len(allCountries))
	for range allCountries {
		res := <-ch
		if res.err == nil {
			m[res.country] = res.td
		} else if res.err == notFoundErr {
			m[res.country] = &tracklistData{}
		} else {
			return nil, fmt.Errorf("%v: %v", res.country, res.err)
		}
	}
	return m, nil
}

// albumData is the toplevel object returned by /v1/albums/<id>.
type albumData struct {
	ID             int          `json:"id"`
	Title          string       `json:"title"` // album title
	AllowStreaming bool         `json:"allowStreaming"`
	NumberOfTracks int          `json:"numberOfTracks"`
	ReleaseDate    jsonDate     `json:"releaseDate"` // e.g. "2016-06-24"
	Copyright      string       `json:"copyright"`
	Type           string       `json:"type"`  // "ALBUM", "EP", "SINGLE"
	Cover          string       `json:"cover"` // UUID
	UPC            string       `json:"upc"`
	Artist         artistData   `json:"artist"`
	Artists        []artistData `json:"artists"`
}

type artistData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // e.g. "MAIN", "FEATURED"
}

// creditsData is the toplevel object returned by /v1/albums/<id>/credits.
type creditsData []struct {
	Type         string `json:"type"` // e.g. "Producer", "Composer", "Record Label", etc.
	Contributors []struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	} `json:"contributors"`
}

// tracklistData is the toplevel object returned by /v1/albums/<id>/tracks.
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
