// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/url"
	"strconv"
)

// URL holds data used to seed forms with an entity's relationship to a URL.
type URL struct {
	// URL contains the full URL.
	URL string
	// LinkType contains the link type ID.
	// Applicable link types should end in "<Entity>_URL_Link", depending on the
	// type of the entity being linked to the URL (but note that the LinkType
	// enum may not include all possible values).
	LinkType LinkType
}

// setParams sets query parameters in vals corresponding to non-empty fields in url.
// The supplied prefix (e.g. "urls.0.") is prepended before each parameter name.
func (url *URL) setParams(vals url.Values, prefix string) {
	if url.URL != "" {
		vals.Set(prefix+"url", url.URL)
	}
	if url.LinkType != 0 {
		vals.Set(prefix+"link_type", strconv.Itoa(int(url.LinkType)))
	}
}
