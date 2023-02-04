// Copyright 2023 Daniel Erat.
// All rights reserved.

// Package internal defines internal types for online sources.
package internal

import (
	"context"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/web"
)

// Provider gets information from an online music provider.
type Provider interface {
	// CleanURL returns a normalized version of the supplied URL.
	// An error is returned if the URL doesn't match a supported format for the provider.
	CleanURL(orig string) (string, error)
	// NeedsPage returns true if Release needs the page to be fetched and passed to it.
	// Providers should return false if they use some other API to get data.
	NeedsPage() bool
	// Release extracts release information from the supplied page.
	// If NeedsPage returns false, then the supplied page will be nil.
	// The img return value is nil if a cover image is not found.
	Release(ctx context.Context, page *web.Page, url string, db *db.DB, cfg *Config) (
		rel *seed.Release, img *seed.Info, err error)
	// ExampleURL returns an example URL that can be displayed to the user.
	ExampleURL() string
}

// Config is passed to Provider implementations to configure their behavior.
type Config struct {
	// ExtractTrackArtists indicates that artist names should be extracted from the
	// beginnings of track names, e.g. "Artist - Title". This is currently only used
	// for Bandcamp pages.
	ExtractTrackArtists bool
	// CountryCode contains the country that should be used when requesting album
	// data, e.g. "US" or "DE". This is currently only used for the Tidal API.
	CountryCode string
	// DisallowNetwork indicates that network requests should not be made.
	// This can be set by tests.
	DisallowNetwork bool
}
