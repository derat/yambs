// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"github.com/derat/yambs/seed"
)

// releaseFields defines fields that can be set in a seed.Release.
var releaseFields = map[string]fieldInfo{
	"title": {
		"Release title",
		func(r *seed.Release, k, v string) error { return setString(&r.Title, v) },
	},
	"release_group": {
		"MBID of existing release group",
		func(r *seed.Release, k, v string) error { return setMBID(&r.ReleaseGroup, v) },
	},
	"types": {
		`Comma-separated [types](` + rgTypeURL + `) for new release group (e.g. "Single,Soundtrack")`,
		func(r *seed.Release, k, v string) error {
			var vals []string
			setStringSlice(&vals, v, ",")
			for _, v := range vals {
				r.Types = append(r.Types, seed.ReleaseGroupType(v))
			}
			return nil
		},
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
		`Release barcode (or "none" if release has no barcode)`,
		func(r *seed.Release, k, v string) error { return setString(&r.Barcode, v) },
	},
	"language": {
		`Release language as [ISO 693-3 code](` + langURL + `) (e.g. "eng", "deu", "jpn")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Language, v) },
	},
	"script": {
		`Release script as [ISO 15924 code](` + scriptURL + `) (e.g. "Latn", "Cyrl")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Script, v) },
	},
	"status": {
		`[Release status](` + statusURL + `) (e.g. "Official", "Promotion", "Bootleg", "Pseudo-Release")`,
		func(r *seed.Release, k, v string) error { return setString((*string)(&r.Status), v) },
	},
	"packaging": {
		`[Release packaging](` + packagingURL + `) (e.g. "Jewel Case", "None")`,
		func(r *seed.Release, k, v string) error { return setString((*string)(&r.Packaging), v) },
	},
	"event*_year": {
		"Year of release event",
		func(rel *seed.Release, k, v string) error {
			return releaseEvent(rel, k, func(ev *seed.ReleaseEvent) error {
				return setInt(&ev.Year, v)
			})
		},
	},
	"event*_month": {
		"Month of release event (1-12)",
		func(rel *seed.Release, k, v string) error {
			return releaseEvent(rel, k, func(ev *seed.ReleaseEvent) error {
				return setInt(&ev.Month, v)
			})
		},
	},
	"event*_day": {
		"Day of release event (1-31)",
		func(rel *seed.Release, k, v string) error {
			return releaseEvent(rel, k, func(ev *seed.ReleaseEvent) error {
				return setInt(&ev.Day, v)
			})
		},
	},
	"event*_date": {
		`Date of release event as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(rel *seed.Release, k, v string) error {
			return releaseEvent(rel, k, func(ev *seed.ReleaseEvent) error {
				var err error
				ev.Year, ev.Month, ev.Day, err = parseDate(v)
				return err
			})
		},
	},
	"event*_country": {
		`Country of release event as [ISO 3166-1 alpha-2 code](` + countryURL + `) (e.g. "GB", "US", "FR", "XW")`,
		func(rel *seed.Release, k, v string) error {
			return releaseEvent(rel, k, func(ev *seed.ReleaseEvent) error {
				return setString(&ev.Country, v)
			})
		},
	},
	"label*_mbid": {
		"MBID of label",
		func(rel *seed.Release, k, v string) error {
			return releaseLabel(rel, k, func(lab *seed.ReleaseLabel) error {
				return setMBID(&lab.MBID, v)
			})
		},
	},
	"label*_catalog": {
		"Catalog number for label",
		func(rel *seed.Release, k, v string) error {
			return releaseLabel(rel, k, func(lab *seed.ReleaseLabel) error {
				return setString(&lab.CatalogNumber, v)
			})
		},
	},
	"label*_name": {
		"Name for label (to prefill search if MBID is unknown)",
		func(rel *seed.Release, k, v string) error {
			return releaseLabel(rel, k, func(lab *seed.ReleaseLabel) error {
				return setString(&lab.Name, v)
			})
		},
	},
	"mbid": {
		"MBID of existing release to edit (if empty, create release)",
		func(rel *seed.Release, k, v string) error { return setMBID(&rel.MBID, v) },
	},
	"medium*_format": {
		`[Medium format](` + formatURL + `) (e.g. "CD", "Digital Media")`,
		func(rel *seed.Release, k, v string) error {
			return releaseMedium(rel, k, func(med *seed.Medium) error {
				return setString((*string)(&med.Format), v)
			})
		},
	},
	"medium*_name": {
		"Medium name",
		func(rel *seed.Release, k, v string) error {
			return releaseMedium(rel, k, func(med *seed.Medium) error {
				return setString(&med.Name, v)
			})
		},
	},
	"medium*_track*_title": {
		"Track title",
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrack(rel, k, func(tr *seed.Track) error {
				return setString(&tr.Title, v)
			})
		},
	},
	"medium*_track*_number": {
		"Track number",
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrack(rel, k, func(tr *seed.Track) error {
				return setString(&tr.Number, v)
			})
		},
	},
	"medium*_track*_recording": {
		"Track recording MBID",
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrack(rel, k, func(tr *seed.Track) error {
				return setMBID(&tr.Recording, v)
			})
		},
	},
	"medium*_track*_length": {
		`Track length as e.g. "3:45.01" or total milliseconds`,
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrack(rel, k, func(tr *seed.Track) error {
				return setDuration(&tr.Length, v)
			})
		},
	},
	"edit_note": {
		"Note attached to edit",
		func(r *seed.Release, k, v string) error { return setString(&r.EditNote, v) },
	},
}

func init() {
	// Add common fields.
	addArtistCreditFields(releaseFields, "",
		func(fn artistFunc) interface{} {
			return func(r *seed.Release, k, v string) error {
				return indexedField(&r.Artists, k, "artist",
					func(ac *seed.ArtistCredit) error { return fn(ac, v) })
			}
		})
	addURLFields(releaseFields,
		func(fn urlFunc) interface{} {
			return func(r *seed.Release, k, v string) error {
				return indexedField(&r.URLs, k, "url",
					func(url *seed.URL) error { return fn(url, v) })
			}
		})
	addArtistCreditFields(releaseFields, "medium*_track*_",
		func(fn artistFunc) interface{} {
			return func(r *seed.Release, k, v string) error {
				return releaseMediumTrackArtist(r, k,
					func(ac *seed.ArtistCredit) error { return fn(ac, v) })
			}
		})
}

// Helper functions to make code for setting indexed fields slightly less horrendous.
func releaseEvent(r *seed.Release, k string, fn func(*seed.ReleaseEvent) error) error {
	return indexedField(&r.Events, k, "event", fn)
}
func releaseLabel(r *seed.Release, k string, fn func(*seed.ReleaseLabel) error) error {
	return indexedField(&r.Labels, k, "label", fn)
}
func releaseMedium(r *seed.Release, k string, fn func(*seed.Medium) error) error {
	return indexedField(&r.Mediums, k, "medium", fn)
}
func releaseMediumTrack(r *seed.Release, k string, fn func(*seed.Track) error) error {
	return releaseMedium(r, k, func(m *seed.Medium) error {
		return indexedField(&m.Tracks, k, `^medium\d*_track`, fn)
	})
}
func releaseMediumTrackArtist(r *seed.Release, k string, fn func(*seed.ArtistCredit) error) error {
	return releaseMediumTrack(r, k, func(t *seed.Track) error {
		return indexedField(&t.Artists, k, `^medium\d*_track\d*_artist`, fn)
	})
}
