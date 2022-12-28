// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package online generates seeded edits from online music providers.
package online

import (
	"context"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online/bandcamp"
	"github.com/derat/yambs/sources/text"
	"github.com/derat/yambs/web"
)

// editNote is appended to automatically-generated edit notes.
const editNote = "\n\n(seeded using https://github.com/derat/yambs)"

// CleanURL returns a normalized version of the supplied URL.
// An error is returned if the URL doesn't match a known format (but note that it may still be
// possible to handle the page, e.g. if it's a Bandcamp album being served from a custom domain).
func CleanURL(orig string) (string, error) {
	return bandcamp.CleanURL(orig)
}

// Fetch generates seeded edits from the page at url.
func Fetch(ctx context.Context, url string, rawSetCmds []string, db *db.DB) ([]seed.Edit, error) {
	setCmds, err := text.ParseSetCommands(rawSetCmds, seed.ReleaseType)
	if err != nil {
		return nil, err
	}

	page, err := web.FetchPage(ctx, url)
	if err != nil {
		return nil, err
	}
	rel, img, err := bandcamp.Release(ctx, page, url, db)
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
