// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"net/http"
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
// method contains the HTTP method that will be used (e.g. "GET" or "POST").
func (url *URL) setParams(vals url.Values, prefix, method string) {
	// Weirdly, recordings (or maybe all forms seeded with GET) use different
	// field names for URLs from the ones that are documented at
	// https://wiki.musicbrainz.org/Development/Release_Editor_Seeding
	// (which uses POST). The place where I finally found the alternate names is
	// https://github.com/metabrainz/musicbrainz-server/blob/master/root/static/scripts/edit/externalLinks.js:
	//
	//  const seededLinkRegex = new RegExp(
	//    '(?:\\?|&)edit-' + sourceType +
	//      '\\.url\\.([0-9]+)\\.(text|link_type_id)=([^&]+)',
	//    'g',
	//  );
	ifGet := func(get, post string) string {
		if method == http.MethodGet {
			return get
		}
		return post
	}

	if url.URL != "" {
		vals.Set(prefix+ifGet("text", "url"), url.URL)
	}
	if url.LinkType != 0 {
		vals.Set(prefix+ifGet("link_type_id", "link_type"),
			strconv.Itoa(int(url.LinkType)))
	}
}
