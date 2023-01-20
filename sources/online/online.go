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
	"github.com/derat/yambs/sources/online/qobuz"
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
func Fetch(ctx context.Context, url string, rawSetCmds []string, db *db.DB) ([]seed.Edit, error) {
	setCmds, err := text.ParseSetCommands(rawSetCmds, seed.ReleaseEntity)
	if err != nil {
		return nil, err
	}

	page, err := web.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}
	rel, img, err := selectProvider(url).Release(ctx, page, url, db, true /* network */)
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

// Provider gets information from an online music provider.
type Provider interface {
	// CleanURL returns a normalized version of the supplied URL.
	// An error is returned if the URL doesn't match a supported format for the provider.
	CleanURL(orig string) (string, error)
	// Release extracts release information from the supplied page.
	// The img return value is nil if a cover image is not found.
	Release(ctx context.Context, page *web.Page, url string, db *db.DB, network bool) (
		rel *seed.Release, img *seed.Info, err error)
	// ExampleURL returns an example URL that can be displayed to the user.
	ExampleURL() string
}

var allProviders = []Provider{
	&bandcamp.Provider{},
	&qobuz.Provider{},
}

// ExampleURLs holds example URLs that can be displayed to the user.
var ExampleURLs []string

func init() {
	for _, p := range allProviders {
		ExampleURLs = append(ExampleURLs, p.ExampleURL())
	}
}

// selectProvider chooses the appropriate provider for handling url.
func selectProvider(url string) Provider {
	for _, p := range allProviders {
		if _, err := p.CleanURL(url); err == nil {
			return p
		}
	}
	// Fall back to Bandcamp to support custom domains.
	return &bandcamp.Provider{}
}
