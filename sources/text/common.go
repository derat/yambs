// Copyright 2023 Daniel Erat.
// All rights reserved.

package text

import (
	"errors"

	"github.com/derat/yambs/seed"
)

// These functions update the supplied structs with the supplied values.
type artistFunc func(ac *seed.ArtistCredit, v string) error
type relFunc func(rel *seed.Relationship, k, v string) error
type urlFunc func(url *seed.URL, v string) error

// addArtistCreditFields adds "artist*_"-prefixed fields for seed.ArtistCredit.
// prefix is prepended to the field names.
// fn should return an appropriately-typed fieldInfo.Fn that invokes the artistFunc
// with the seed.ArtistCredit and user-supplied value.
func addArtistCreditFields(fields map[string]fieldInfo, prefix string, fn func(artistFunc) interface{}) {
	fields[prefix+"artist*_mbid"] = fieldInfo{
		"Artist's MBID",
		fn(func(ac *seed.ArtistCredit, v string) error {
			return setMBID(&ac.MBID, v) // converted to database ID by Finish
		}),
	}
	fields[prefix+"artist*_name"] = fieldInfo{
		"Artist's name for database search",
		fn(func(ac *seed.ArtistCredit, v string) error { return setString(&ac.Name, v) }),
	}
	fields[prefix+"artist*_credited"] = fieldInfo{
		"Artist's name as credited",
		fn(func(ac *seed.ArtistCredit, v string) error { return setString(&ac.NameAsCredited, v) }),
	}
	fields[prefix+"artist*_join"] = fieldInfo{
		`Join phrase used to separate artist and next artist (e.g. " & ")`,
		fn(func(ac *seed.ArtistCredit, v string) error { return setString(&ac.JoinPhrase, v) }),
	}
}

// addRelationshipFields adds "rel*_"-prefixed fields for seed.Relationship.
// fn should return an appropriately-typed fieldInfo.Fn that invokes the relFunc
// with the seed.Relationship and user-supplied key and value.
func addRelationshipFields(fields map[string]fieldInfo, fn func(relFunc) interface{}) {
	fields["rel*_backward"] = fieldInfo{
		`Whether the relationship direction is reversed ("1" or "true" if true)`,
		fn(func(rel *seed.Relationship, k, v string) error { return setBool(&rel.Backward, v) }),
	}
	fields["rel*_begin_date"] = fieldInfo{
		`Date when relationship began as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		fn(func(rel *seed.Relationship, k, v string) error { return setDate(&rel.BeginDate, v) }),
	}
	fields["rel*_end_date"] = fieldInfo{
		`Date when relationship ended as "YYYY-MM-DD", "YYYY-MM", or "YYYY"`,
		fn(func(rel *seed.Relationship, k, v string) error { return setDate(&rel.EndDate, v) }),
	}
	fields["rel*_ended"] = fieldInfo{
		`Whether the relationship has ended ("1" or "true" if true)`,
		fn(func(rel *seed.Relationship, k, v string) error { return setBool(&rel.Ended, v) }),
	}
	fields["rel*_source_credit"] = fieldInfo{
		"Source entity's name as credited in relationship",
		fn(func(rel *seed.Relationship, k, v string) error { return setString(&rel.SourceCredit, v) }),
	}
	fields["rel*_target"] = fieldInfo{
		"MBID or name of entity at other end of relationship",
		fn(func(rel *seed.Relationship, k, v string) error { return setString(&rel.Target, v) }),
	}
	fields["rel*_target_credit"] = fieldInfo{
		"Target entity's name as credited in relationship",
		fn(func(rel *seed.Relationship, k, v string) error { return setString(&rel.TargetCredit, v) }),
	}
	fields["rel*_type"] = fieldInfo{
		"Integer [link type](" + linkTypeURL + ") or UUID describing the relationship type",
		fn(func(rel *seed.Relationship, k, v string) error {
			if err := setInt((*int)(&rel.Type), v); err != nil {
				if err := setMBID(&rel.TypeUUID, v); err != nil {
					return errors.New("not integer or UUID")
				}
			}
			return nil
		}),
	}

	// attrFn is an adapted version of fn that returns a fieldInfo.Fn that invokes afn
	// with a RelationshipAttribute. Thank goodness for type-checking, since there's
	// no way that I'd get this mess right otherwise.
	attrFn := func(afn func(*seed.RelationshipAttribute, string) error) interface{} {
		return fn(func(r *seed.Relationship, k, v string) error {
			return indexedField(&r.Attributes, k, `^rel\d*_attr`,
				func(attr *seed.RelationshipAttribute) error { return afn(attr, v) })
		})
	}

	fields["rel*_attr*_credited"] = fieldInfo{
		"Relationship attribute's credited-as text",
		attrFn(func(attr *seed.RelationshipAttribute, v string) error { return setString(&attr.CreditedAs, v) }),
	}
	fields["rel*_attr*_text"] = fieldInfo{
		"Relationship attribute's additional text",
		attrFn(func(attr *seed.RelationshipAttribute, v string) error { return setString(&attr.TextValue, v) }),
	}
	fields["rel*_attr*_type"] = fieldInfo{
		"Integer [link attribute type](" + linkAttrTypeURL + ") or UUID describing the relationship attribute type",
		attrFn(func(attr *seed.RelationshipAttribute, v string) error {
			if err := setInt((*int)(&attr.Type), v); err != nil {
				if err := setMBID(&attr.TypeUUID, v); err != nil {
					return errors.New("not integer or UUID")
				}
			}
			return nil
		}),
	}
}

// addURLFields adds "url*_"-prefixed fields for seed.URL.
// fn should return an appropriately-typed fieldInfo.Fn that invokes the urlFunc
// with the seed.URL and user-supplied value.
func addURLFields(fields map[string]fieldInfo, fn func(urlFunc) interface{}) {
	fields["url*_url"] = fieldInfo{
		"URL related to entity",
		fn(func(u *seed.URL, v string) error { return setString(&u.URL, v) }),
	}
	fields["url*_type"] = fieldInfo{
		"Integer [link type](" + linkTypeURL + ") describing how URL is related to entity",
		fn(func(u *seed.URL, v string) error { return setInt((*int)(&u.LinkType), v) }),
	}
}
