// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package online generates seeded edits from online music providers.
package online

import (
	"context"
	"errors"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online/bandcamp"
	"github.com/derat/yambs/sources/online/internal"
	"github.com/derat/yambs/sources/online/qobuz"
	"github.com/derat/yambs/sources/online/tidal"
	"github.com/derat/yambs/sources/text"
	"github.com/derat/yambs/web"
)

// editNote is appended to automatically-generated edit notes.
const editNote = "\n\n(seeded using https://github.com/derat/yambs)"

// CleanURL returns a normalized version of the supplied URL.
// An error is returned if the URL doesn't match a known format (but note that it may still be
// possible to handle the page using Fetch, e.g. if it's a Bandcamp album being served from a custom
// domain).
func CleanURL(orig string) (string, error) {
	for _, p := range allProviders {
		if cleaned, err := p.CleanURL(orig); err == nil {
			return cleaned, nil
		}
	}
	return "", errors.New("unsupported URL")
}

// Fetch generates seeded edits from the page at url.
// If cfg is nil, a default configuration will be used.
func Fetch(ctx context.Context, url string, rawSetCmds []string,
	db *db.DB, cfg *Config) ([]seed.Edit, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	setCmds, err := text.ParseSetCommands(rawSetCmds, seed.ReleaseEntity)
	if err != nil {
		return nil, err
	}

	prov := selectProvider(url)
	if prov == nil {
		return nil, errors.New("no suitable provider found")
	}
	var page *web.Page
	if prov.NeedsPage() {
		if page, err = web.FetchPage(ctx, url); err != nil {
			return nil, err
		}
	}
	rel, img, err := prov.Release(ctx, page, url, db, (*internal.Config)(cfg))
	if err != nil {
		return nil, err
	}
	rel.EditNote = url + editNote

	for _, cmd := range setCmds {
		if err := text.SetField(rel, cmd[0], cmd[1]); err != nil {
			return nil, err
		}
	}
	edits := []seed.Edit{rel}

	if img != nil {
		edits = append(edits, img)
		rel.RedirectURI = seed.AddCoverArtRedirectURI
	}
	return edits, nil
}

// Config configures Fetch's behavior.
// Aliasing an internal type like this is weird, but it avoids a circular dependency
// (subpackages can't depend on this package since it depends on the subpackages).
type Config internal.Config

var allProviders = []internal.Provider{
	&bandcamp.Provider{},
	&qobuz.Provider{},
	&tidal.Provider{},
}

// ExampleURLs holds example URLs that can be displayed to the user.
var ExampleURLs []string

func init() {
	for _, p := range allProviders {
		ExampleURLs = append(ExampleURLs, p.ExampleURL())
	}
}

// selectProvider chooses the appropriate provider for handling url.
func selectProvider(url string) internal.Provider {
	for _, p := range allProviders {
		if _, err := p.CleanURL(url); err == nil {
			return p
		}
	}
	// Fall back to Bandcamp to support custom domains.
	return &bandcamp.Provider{}
}
