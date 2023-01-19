// Copyright 2022 Daniel Erat.
// All rights reserved.

package text

import (
	"errors"

	"github.com/derat/yambs/seed"
)

// recordingFields defines fields that can be set in a seed.Recording.
var recordingFields = map[string]fieldInfo{
	"artist": {
		"MBID of artist receiving primary credit for recording",
		func(r *seed.Recording, k, v string) error { return setMBID(&r.Artist, v) },
	},
	"artist*_mbid": {
		"Artist's MBID",
		func(r *seed.Recording, k, v string) error {
			return recordingArtist(r, k, func(ac *seed.ArtistCredit) error {
				return setMBID(&ac.MBID, v) // converted to database ID by Finish
			})
		},
	},
	"artist*_name": {
		"Artist's name for database search",
		func(r *seed.Recording, k, v string) error {
			return recordingArtist(r, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.Name, v)
			})
		},
	},
	"artist*_credited": {
		"Artist's name as credited",
		func(r *seed.Recording, k, v string) error {
			return recordingArtist(r, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.NameAsCredited, v)
			})
		},
	},
	"artist*_join": {
		`Join phrase used to separate artist and next artist (e.g. " & ")`,
		func(r *seed.Recording, k, v string) error {
			return recordingArtist(r, k, func(ac *seed.ArtistCredit) error {
				return setString(&ac.JoinPhrase, v)
			})
		},
	},
	"disambiguation": {
		"Comment disambiguating this recording from others with similar names",
		func(r *seed.Recording, k, v string) error { return setString(&r.Disambiguation, v) },
	},
	"edit_note": {
		"Note attached to edit",
		func(r *seed.Recording, k, v string) error { return setString(&r.EditNote, v) },
	},
	"isrcs": {
		"Comma-separated ISRCs identifying recording",
		func(r *seed.Recording, k, v string) error { return setStringSlice(&r.ISRCs, v, ",") },
	},
	"length": {
		`Recording's duration as e.g. "3:45.01" or total milliseconds`,
		func(r *seed.Recording, k, v string) error { return setDuration(&r.Length, v) },
	},
	"mbid": {
		"MBID of existing recording to edit (if empty, create recording)",
		func(r *seed.Recording, k, v string) error { return setMBID(&r.MBID, v) },
	},
	"name": {
		"Recording's name",
		func(r *seed.Recording, k, v string) error { return setString(&r.Name, v) },
	},
	"rel*_backward": {
		`Whether the relationship direction is reversed ("1" or "true" if true)`,
		func(r *seed.Recording, k, v string) error {
			return recordingRelationship(r, k, func(rel *seed.Relationship) error { return setBool(&rel.Backward, v) })
		},
	},
	"rel*_begin_date": {
		`Date when relationship began as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(r *seed.Recording, k, v string) error {
			return recordingRelationship(r, k, func(rel *seed.Relationship) error {
				var err error
				rel.BeginYear, rel.BeginMonth, rel.BeginDay, err = parseDate(v)
				return err
			})
		},
	},
	"rel*_end_date": {
		`Date when relationship ended as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		func(r *seed.Recording, k, v string) error {
			return recordingRelationship(r, k, func(rel *seed.Relationship) error {
				var err error
				rel.EndYear, rel.EndMonth, rel.EndDay, err = parseDate(v)
				return err
			})
		},
	},
	"rel*_ended": {
		`Whether the relationship has ended ("1" or "true" if true)`,
		func(r *seed.Recording, k, v string) error {
			return recordingRelationship(r, k, func(rel *seed.Relationship) error { return setBool(&rel.Ended, v) })
		},
	},
	"rel*_target": {
		"MBID or name of entity at other end of relationship",
		func(r *seed.Recording, k, v string) error {
			return recordingRelationship(r, k, func(rel *seed.Relationship) error { return setString(&rel.Target, v) })
		},
	},
	"rel*_type": {
		"Integer [link type](" + linkTypeURL + ") or UUID describing the relationship type",
		func(r *seed.Recording, k, v string) error {
			return recordingRelationship(r, k, func(rel *seed.Relationship) error {
				if err := setInt((*int)(&rel.Type), v); err != nil {
					if err := setMBID(&rel.TypeUUID, v); err != nil {
						return errors.New("not integer or UUID")
					}
				}
				return nil
			})
		},
	},
	"rel*_attr*_credited": {
		"Relationship attribute's credited-as text",
		func(r *seed.Recording, k, v string) error {
			return recordingRelationshipAttribute(r, k, func(attr *seed.RelationshipAttribute) error {
				return setString(&attr.CreditedAs, v)
			})
		},
	},
	"rel*_attr*_text": {
		"Relationship attribute's additional text",
		func(r *seed.Recording, k, v string) error {
			return recordingRelationshipAttribute(r, k, func(attr *seed.RelationshipAttribute) error {
				return setString(&attr.TextValue, v)
			})
		},
	},
	"rel*_attr*_type": {
		"Integer [link attribute type](" + linkAttrTypeURL + ") or UUID describing the relationship attribute type",
		func(r *seed.Recording, k, v string) error {
			return recordingRelationshipAttribute(r, k, func(attr *seed.RelationshipAttribute) error {
				if err := setInt((*int)(&attr.Type), v); err != nil {
					if err := setMBID(&attr.TypeUUID, v); err != nil {
						return errors.New("not integer or UUID")
					}
				}
				return nil
			})
		},
	},
	"url*_url": {
		"URL related to recording",
		func(r *seed.Recording, k, v string) error {
			return recordingURL(r, k, func(u *seed.URL) error { return setString(&u.URL, v) })
		},
	},
	"url*_type": {
		"Integer [link type](" + linkTypeURL + ") describing how URL is related to recording",
		func(r *seed.Recording, k, v string) error {
			return recordingURL(r, k, func(u *seed.URL) error { return setInt((*int)(&u.LinkType), v) })
		},
	},
	"video": {
		`Whether this is a video recording ("1" or "true" if true)`,
		func(r *seed.Recording, k, v string) error { return setBool(&r.Video, v) },
	},
}

func recordingArtist(r *seed.Recording, k string, fn func(*seed.ArtistCredit) error) error {
	return indexedField(&r.Artists, k, "artist", fn)
}
func recordingURL(r *seed.Recording, k string, fn func(*seed.URL) error) error {
	return indexedField(&r.URLs, k, "url", fn)
}
func recordingRelationship(r *seed.Recording, k string, fn func(*seed.Relationship) error) error {
	return indexedField(&r.Relationships, k, "rel", fn)
}
func recordingRelationshipAttribute(r *seed.Recording, k string, fn func(*seed.RelationshipAttribute) error) error {
	return recordingRelationship(r, k, func(r *seed.Relationship) error {
		return indexedField(&r.Attributes, k, `^rel\d*_attr`, fn)
	})
}
