// Copyright 2023 Daniel Erat.
// All rights reserved.

package seed

import (
	"fmt"
	"net/url"
	"strconv"
)

// Relationship holds data used to seed forms with non-URL relationships between entities.
// See https://musicbrainz.org/doc/Relationships.
type Relationship struct {
	// Target contains the MBID or name of the entity at the other end of the relationship.
	// If a name is supplied rather than an MBID, the field will be seeded for a database search.
	Target string
	// Type contains the database ID of the relationship type.
	// The relationship will be ignored if the link type is inappropriate for the type of the
	// entity being edited (e.g. when seeding a recording, a LinkType_*_Recording type should
	// be specified).
	Type LinkType
	// TypeUUID contains the UUID of the relationship type. It is only used if Type is unset.
	// UUIDs can be found at https://musicbrainz.org/relationships.
	TypeUUID string
	// Attributes contains additional attributes associated with this relationship.
	Attributes []RelationshipAttribute
	// BeginYear contains the year when the relationship started, or 0 if unknown.
	BeginYear int
	// BeginMonth contains the 1-indexed month when the relationship started, or 0 if unknown.
	BeginMonth int
	// BeginDay contains the day of the month when the relationship started, or 0 if unknown.
	BeginDay int
	// EndYear contains the year when the relationship ended, or 0 if unknown.
	EndYear int
	// EndMonth contains the 1-indexed month when the relationship ended, or 0 if unknown.
	EndMonth int
	// EndDay contains the day of the month when the relationship ended, or 0 if unknown.
	EndDay int
	// Ended describes whether the relationship has ended.
	Ended bool
	// Backward describes whether the relationship direction should be reversed.
	// For example, when using LinkType_SamplesMaterial_Recording_Recording,
	// a true value indicates that Target sampled the seeded recording rather than
	// the seeded recording sampling Target.
	Backward bool
}

// setParams sets query parameters in vals corresponding to non-empty fields in rel.
// The supplied prefix (e.g. "rels.0.") is prepended before each parameter name.
func (rel *Relationship) setParams(vals url.Values, prefix string) {
	if rel.Target != "" {
		vals.Set(prefix+"target", rel.Target)
	}

	// The "type" parameter appears to accept both database IDs and UUIDs.
	if rel.Type != 0 {
		vals.Set(prefix+"type", strconv.Itoa(int(rel.Type)))
	} else if rel.TypeUUID != "" {
		vals.Set(prefix+"type", rel.TypeUUID)
	}

	for i, attr := range rel.Attributes {
		attr.setParams(vals, prefix+fmt.Sprintf("attributes.%d.", i))
	}

	// https://bitbucket.org/metabrainz/musicbrainz-server/pull-requests/1393 says
	// "To support partial dates it accepts hyphens in place of the missing parts,
	// e.g. --09-09 for a missing year or 1999---09 for a missing month."
	// This doesn't seem to work, though (which seems fine, since it's pretty weird!).
	setDate := func(name string, y, m, d int) {
		var s string
		if y > 0 {
			s += fmt.Sprintf("%04d", y)
			if m > 0 {
				s += fmt.Sprintf("-%02d", m)
				if d > 0 {
					s += fmt.Sprintf("-%02d", d)
				}
			}
		}
		if s != "" {
			vals.Set(prefix+name, s)
		}
	}
	setDate("begin_date", rel.BeginYear, rel.BeginMonth, rel.BeginDay)
	setDate("end_date", rel.EndYear, rel.EndMonth, rel.EndDay)

	if rel.Ended {
		vals.Set(prefix+"ended", "1")
	}
	if rel.Backward {
		// Note that https://github.com/metabrainz/musicbrainz-server/commit/68841055a5fc
		// seems this to have changed this from the "direction=backward" described in
		// https://bitbucket.org/metabrainz/musicbrainz-server/pull-requests/1393.
		vals.Set(prefix+"backward", "1")
	}
}

// RelationshipAttribute modifies a relationship between two entities.
type RelationshipAttribute struct {
	// Type contains the database ID of the attribute type.
	Type LinkAttributeType
	// TypeUUID contains the UUID of the attribute type. It is only used if Type is unset.
	// UUIDs can be found at https://musicbrainz.org/relationship-attributes.
	TypeUUID string
	// CreditedAs is used to fill the "credited as" field (e.g. describing how an instrument was credited).
	CreditedAs string
	// TextValue holds an additional text value associated with the relationship.
	// TODO: Document what this is used for. It obviously ends up in the link_attribute_text_value
	// table, but I haven't found details anywhere. I suspect that it might be used for things like
	// LinkAttributeType_Time.
	TextValue string
}

// setParams sets query parameters in vals corresponding to non-empty fields in attr.
// The supplied prefix (e.g. "rels.0.attributes.0") is prepended before each parameter name.
func (attr *RelationshipAttribute) setParams(vals url.Values, prefix string) {
	if attr.Type != 0 {
		vals.Set(prefix+"type", strconv.Itoa(int(attr.Type)))
	} else if attr.TypeUUID != "" {
		vals.Set(prefix+"type", attr.TypeUUID)
	}
	if attr.CreditedAs != "" {
		vals.Set(prefix+"credited_as", attr.CreditedAs)
	}
	if attr.TextValue != "" {
		vals.Set(prefix+"text_value", attr.TextValue)
	}
}
