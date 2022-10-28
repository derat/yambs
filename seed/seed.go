// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package seed generates URLs that pre-fill fields when adding entities to MusicBrainz.
package seed

import (
	"net/url"
)

const (
	maxDescLen    = 40 // max length for description components
	mbidPrefixLen = 8
)

// Type describes a type of MusicBrainz edit.
type Type string

const (
	RecordingType Type = "recording"
	ReleaseType   Type = "release"
)

// Edit represents a seeded MusicBrainz edit.
type Edit interface {
	// Type returns the type of entity being edited.
	Type() Type
	// Description returns a human-readable description of the edit.
	Description() string
	// URL returns a URL to seed the edit form.
	URL() string
	// Params returns form values that should be sent to seed the edit form.
	Params() url.Values
	// CanGet() returns true if the request for URL can use the GET method rather than POST.
	// GET is preferable since it avoids an anti-CSRF interstitial page.
	CanGet() bool
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
