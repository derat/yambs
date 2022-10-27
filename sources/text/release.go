// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// releaseFields defines fields that can be set in a seed.Release.
var releaseFields = map[string]fieldInfo{
	"title": {
		"Release's name",
		func(r *seed.Release, k, v string) error { return setString(&r.Title, v) },
	},
	"release_group": {
		"MBID of existing release group",
		func(r *seed.Release, k, v string) error { return setString(&r.ReleaseGroup, v) },
	},
	"types": {
		`Comma-separated types for new release group (e.g. "single,soundtrack")`,
		func(r *seed.Release, k, v string) error { return setStringSlice(&r.Types, v, ",") },
	},
	"disambiguation": {
		"Comment disambiguating this release from others with similar names",
		func(r *seed.Release, k, v string) error { return setString(&r.Disambiguation, v) },
	},
	"annotation": {
		"Additional freeform information describing release",
		func(r *seed.Release, k, v string) error { return setString(&r.Annotation, v) },
	},
	"barcode": {
		`Release's barcode (or "none" if release has no barcode)`,
		func(r *seed.Release, k, v string) error { return setString(&r.Barcode, v) },
	},
	"language": {
		`Release's language as ISO 693-3 code (e.g. "eng", "deu", "jpn")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Language, v) },
	},
	"script": {
		`Release's script as ISO 15924 code (e.g. "Latn", "Cyrl")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Script, v) },
	},
	"status": {
		`Release's status (e.g. "official", "promotion", "bootleg", "pseudo-release")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Status, v) },
	},
	"packaging": {
		// TODO: Document possible values.
		`Release's packaging`,
		func(r *seed.Release, k, v string) error { return setString(&r.Packaging, v) },
	},
	"event*_year": {
		"Year of 0-indexed release event",
		func(r *seed.Release, k, v string) error {
			ev, err := getIndexedField(&r.ReleaseEvents, k, eventIndexRegexp, maxReleaseEvents)
			if err != nil {
				return err
			}
			return setInt(&ev.Year, v)
		},
	},
	"event*_month": {
		"Month of 0-indexed release event",
		func(r *seed.Release, k, v string) error {
			ev, err := getIndexedField(&r.ReleaseEvents, k, eventIndexRegexp, maxReleaseEvents)
			if err != nil {
				return err
			}
			return setInt(&ev.Month, v)
		},
	},
	"event*_day": {
		"Day of 0-indexed release event",
		func(r *seed.Release, k, v string) error {
			ev, err := getIndexedField(&r.ReleaseEvents, k, eventIndexRegexp, maxReleaseEvents)
			if err != nil {
				return err
			}
			return setInt(&ev.Day, v)
		},
	},
	"event*_date": {
		`Date of 0-indexed release event as "YYYY-MM-DD"`,
		func(r *seed.Release, k, v string) error {
			ev, err := getIndexedField(&r.ReleaseEvents, k, eventIndexRegexp, maxReleaseEvents)
			if err != nil {
				return err
			}
			t, err := parseDate(v)
			if err != nil {
				return err
			}
			ev.Year = t.Year()
			ev.Month = int(t.Month())
			ev.Day = t.Day()
			return nil
		},
	},
	"event*_country": {
		`Country of 0-indexed release event as ISO code (e.g. "GB", "US", "FR")`,
		func(r *seed.Release, k, v string) error {
			ev, err := getIndexedField(&r.ReleaseEvents, k, eventIndexRegexp, maxReleaseEvents)
			if err != nil {
				return err
			}
			return setString(&ev.Country, v)
		},
	},
	"label*_mbid": {
		"MBID of 0-indexed label",
		func(r *seed.Release, k, v string) error {
			rl, err := getIndexedField(&r.ReleaseLabels, k, labelIndexRegexp, maxReleaseLabels)
			if err != nil {
				return err
			}
			return setString(&rl.MBID, v)
		},
	},
	"label*_catalog": {
		"Catalog number for 0-indexed label",
		func(r *seed.Release, k, v string) error {
			rl, err := getIndexedField(&r.ReleaseLabels, k, labelIndexRegexp, maxReleaseLabels)
			if err != nil {
				return err
			}
			return setString(&rl.CatalogNumber, v)
		},
	},
	"label*_name": {
		"Name for 0-indexed label (to prefill search if MBID is unknown)",
		func(r *seed.Release, k, v string) error {
			rl, err := getIndexedField(&r.ReleaseLabels, k, labelIndexRegexp, maxReleaseLabels)
			if err != nil {
				return err
			}
			return setString(&rl.Name, v)
		},
	},
	"artist*_mbid": {
		"MBID of 0-indexed artist",
		func(r *seed.Release, k, v string) error {
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.MBID, v)
		},
	},
	"artist*_name": {
		"MusicBrainz name of 0-indexed artist",
		func(r *seed.Release, k, v string) error {
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.Name, v)
		},
	},
	"artist*_credited": {
		"As-credited name of 0-indexed artist",
		func(r *seed.Release, k, v string) error {
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.NameAsCredited, v)
		},
	},
	"artist*_join_phrase": {
		`Join phrase used to separate 0-indexed artist and next artist (e.g. " & ")`,
		func(r *seed.Release, k, v string) error {
			ac, err := getIndexedField(&r.ArtistCredits, k, artistIndexRegexp, maxArtistCredits)
			if err != nil {
				return err
			}
			return setString(&ac.JoinPhrase, v)
		},
	},
	"edit_note": {
		"Note attached to edit",
		func(r *seed.Release, k, v string) error { return setString(&r.EditNote, v) },
	},
}
