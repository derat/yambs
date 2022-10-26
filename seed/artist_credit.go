// Copyright 2022 Daniel Erat.
// All rights reserved.

package seed

import (
	"fmt"
	"net/url"
)

// ArtistCredit holds detailed information about a credited artist.
type ArtistCredit struct {
	// MBID contains the artist entity's MBID, if known.
	// This annoyingly doesn't seem to work for the /recording/create form,
	// so set ID instead in that case (see db.GetArtistID).
	MBID string
	// ID contains the artist's database ID (i.e. the 'id' column from the 'artist' table).
	// This is only needed for the /recording/create form, I think.
	ID int32
	// Name contains the artist's name. This is unneeded if MBID or ID is set.
	Name string
	// NameAsCredited contains the name under which the artist was credited.
	// This is only needed if it's different than MBID or Name.
	NameAsCredited string
	// JoinPhrase contains text for joining this artist's name with the next one's, e.g. " & ".
	JoinPhrase string
}

// setParams sets query parameters in vals corresponding to non-empty fields in ac.
// The supplied prefix (e.g. "artist_credit.names.0.") is prepended before each parameter name.
func (ac *ArtistCredit) setParams(vals url.Values, prefix string) {
	var id string
	if ac.ID > 0 {
		id = fmt.Sprint(ac.ID)
	}
	for k, v := range map[string]string{
		"artist.id":   id,
		"mbid":        ac.MBID,
		"artist.name": ac.Name,
		"name":        ac.NameAsCredited,
		"join_phrase": ac.JoinPhrase,
	} {
		if v != "" {
			vals.Set(prefix+k, v)
		}
	}
}