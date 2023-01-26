// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package seed generates URLs that pre-fill fields when adding entities to MusicBrainz.
package seed

import (
	"context"
	"net/url"

	"github.com/derat/yambs/db"
)

//go:generate go run gen/gen_enums.go

const (
	maxDescLen    = 40 // max length for description components
	mbidPrefixLen = 8
)

// Entity describes a type of MusicBrainz entity being edited.
type Entity string

const (
	LabelEntity     Entity = "label"
	RecordingEntity Entity = "recording"
	ReleaseEntity   Entity = "release"
	WorkEntity      Entity = "work"
	InfoEntity      Entity = "info" // informational edit; not a true entity
)

// EntityTypes lists real database entity types in alphabetical order.
var EntityTypes = []Entity{
	LabelEntity,
	RecordingEntity,
	ReleaseEntity,
	WorkEntity,
}

// Edit represents a seeded MusicBrainz edit.
type Edit interface {
	// Entity returns the type of entity being edited.
	Entity() Entity
	// Description returns a human-readable description of the edit.
	Description() string
	// URL returns a URL to seed the edit form.
	// srv contains the MusicBrainz server hostname, e.g. "musicbrainz.org" or "test.musicbrainz.org".
	URL(srv string) string
	// Params returns form values that should be sent to seed the edit form.
	// Note that some parameters contain multiple values (i.e. don't call Get()).
	Params() url.Values
	// Method() returns the HTTP method that should be used for the request for URL.
	// GET is preferable since it avoids an anti-CSRF interstitial page.
	Method() string
	// Finish fixes up fields in the edit.
	// This should be called once after filling the edit's fields.
	// This only exists because recordings are dumb and require
	// artists' database IDs rather than their MBIDs.
	Finish(ctx context.Context, db *db.DB) error
}

func truncate(orig string, max int, ellide bool) string {
	if len(orig) <= max {
		return orig
	}
	if ellide {
		return orig[:max-1] + "…"
	}
	return orig[:max]
}

func setParams(vals url.Values, m map[string]string, prefix string) {
	for k, v := range m {
		if v != "" {
			vals.Set(prefix+k, v)
		}
	}
}
