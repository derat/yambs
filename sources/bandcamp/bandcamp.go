// Copyright 2022 Daniel Erat.
// All rights reserved.

package bandcamp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/web"
)

// FetchRelease fetches release information from the Bandcamp album page at url.
func FetchRelease(ctx context.Context, url string) (*seed.Release, error) {
	page, err := web.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}
	val, err := page.Query("script[data-tralbum]").Attr("data-tralbum")
	if err != nil {
		return nil, err
	}
	var data albumData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	rel := seed.Release{
		Title:  data.Current.Title,
		Artist: data.Artist,
		Date:   time.Time(data.Current.ReleaseDate),
	}
	return &rel, nil
}

// albumData corresponds to the data-tralbum JSON object embedded in Bandcamp album pages.
type albumData struct {
	Artist  string `json:"artist"`
	Current struct {
		Title       string   `json:"title"`
		ReleaseDate jsonDate `json:"release_date"`
	} `json:"current"`
}

// jsonDate unmarshals a time provided as a JSON string like "07 Oct 2022 00:00:00 GMT".
type jsonDate time.Time

func (d *jsonDate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, err := time.Parse("02 Jan 2006 03:04:05 MST", s)
	*d = jsonDate(t)
	return err
}

func (d jsonDate) String() string { return time.Time(d).String() }
