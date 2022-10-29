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
		func(r *seed.Release, k, v string) error { return setString(&r.ReleaseGroup, v) },
	},
	"types": {
		`Comma-separated types for new release group (e.g. "Single,Soundtrack")`,
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
		`Release language as ISO 693-3 code (e.g. "eng", "deu", "jpn")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Language, v) },
	},
	"script": {
		`Release script as ISO 15924 code (e.g. "Latn", "Cyrl")`,
		func(r *seed.Release, k, v string) error { return setString(&r.Script, v) },
	},
	"status": {
		`Release status (e.g. "official", "promotion", "bootleg", "pseudo-release")`,
		func(r *seed.Release, k, v string) error { return setString((*string)(&r.Status), v) },
	},
	"packaging": {
		`Release packaging (e.g. "Jewel Case", "None"`,
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
		`Date of release event as "YYYY-MM-DD"`,
		func(rel *seed.Release, k, v string) error {
			return releaseEvent(rel, k, func(ev *seed.ReleaseEvent) error {
				t, err := parseDate(v)
				if err != nil {
					return err
				}
				ev.Year = t.Year()
				ev.Month = int(t.Month())
				ev.Day = t.Day()
				return nil
			})
		},
	},
	"event*_country": {
		`Country of release event as ISO code (e.g. "GB", "US", "FR")`,
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
				return setString(&lab.MBID, v)
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
	"artist*_mbid": {
		"Release artist's MBID",
		func(rel *seed.Release, k, v string) error {
			return releaseArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.MBID, v)
			})
		},
	},
	"artist*_name": {
		"Release artist's name for DB search",
		func(rel *seed.Release, k, v string) error {
			return releaseArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.Name, v)
			})
		},
	},
	"artist*_credited": {
		"Release artist's name as credited",
		func(rel *seed.Release, k, v string) error {
			return releaseArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.NameAsCredited, v)
			})
		},
	},
	"artist*_join": {
		`Join phrase separating release artist and next artist (e.g. " & ")`,
		func(rel *seed.Release, k, v string) error {
			return releaseArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.JoinPhrase, v)
			})
		},
	},
	"medium*_format": {
		"Medium format", // TODO: add examples
		func(rel *seed.Release, k, v string) error {
			return releaseMedium(rel, k, func(med *seed.Medium) error {
				return setString(&med.Format, v)
			})
		},
	},
	"medium*_name": {
		"Medium name", // TODO: add examples
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
				return setString(&tr.Recording, v)
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
	"medium*_track*_artist*_mbid": {
		"Track artist's MBID",
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrackArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.MBID, v)
			})
		},
	},
	"medium*_track*_artist*_name": {
		"Track artist's name for database search",
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrackArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.Name, v)
			})
		},
	},
	"medium*_track*_artist*_credited": {
		"Track artist's name as credited",
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrackArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.NameAsCredited, v)
			})
		},
	},
	"medium*_track*_artist*_join": {
		`Join phrase separating track artist and next artist (e.g. " & ")`,
		func(rel *seed.Release, k, v string) error {
			return releaseMediumTrackArtist(rel, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.JoinPhrase, v)
			})
		},
	},
	"url*_url": {
		"URL related to release",
		func(rel *seed.Release, k, v string) error {
			return releaseURL(rel, k, func(u *seed.URL) error { return setString(&u.URL, v) })
		},
	},
	"url*_type": {
		"Integer link type describing how URL is related to release",
		func(rel *seed.Release, k, v string) error {
			return releaseURL(rel, k, func(u *seed.URL) error { return setInt((*int)(&u.LinkType), v) })
		},
	},
	"edit_note": {
		"Note attached to edit",
		func(r *seed.Release, k, v string) error { return setString(&r.EditNote, v) },
	},
}

// Helper functions to make code for setting indexed fields slightly less horrendous.
func releaseEvent(r *seed.Release, k string, fn func(*seed.ReleaseEvent) error) error {
	return indexedField(&r.Events, k, "event", fn)
}
func releaseLabel(r *seed.Release, k string, fn func(*seed.ReleaseLabel) error) error {
	return indexedField(&r.Labels, k, "label", fn)
}
func releaseArtist(r *seed.Release, k string, fn func(*seed.ArtistCredit) error) error {
	return indexedField(&r.Artists, k, "artist", fn)
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
func releaseURL(r *seed.Release, k string, fn func(*seed.URL) error) error {
	return indexedField(&r.URLs, k, "url", fn)
}
