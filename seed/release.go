// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
	"time"
)

// Release holds data used to seed the "Add Release" form at http://musicbrainz.org/release/add.
// See https://musicbrainz.org/doc/Release for more information about releases and
// https://wiki.musicbrainz.org/Development/Release_Editor_Seeding for information about seeding
// this form.
type Release struct {
	// Title contains the release's title.
	Title string
	// Artist contains the name of the artist primarily credited with the release.
	Artist string
	// Date contains the date on which the release was issued.
	Date time.Time
	// TODO: Add a zillion other fields.
}

func (rel *Release) Description() string { return rel.Title + " / " + rel.Artist }

func (rel *Release) URL() string { return "https://musicbrainz.org/release/add" }

func (rel *Release) Params() url.Values {
	vals := make(url.Values)
	if rel.Title != "" {
		vals.Set("name", rel.Title)
	}
	if rel.Artist != "" {
		vals.Set("artist_credit.names.0.name", rel.Artist)
	}
	if !rel.Date.IsZero() {
		vals.Set("events.0.date.year", strconv.Itoa(rel.Date.Year()))
		vals.Set("events.0.date.month", strconv.Itoa(int(rel.Date.Month())))
		vals.Set("events.0.date.day", strconv.Itoa(rel.Date.Day()))
	}
	return vals
}

func (rel *Release) CanGet() bool { return false }
