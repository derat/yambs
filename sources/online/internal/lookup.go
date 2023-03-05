// Copyright 2023 Daniel Erat.
// All rights reserved.

package internal

import (
	"context"
	"log"
	"strings"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/strutil"
)

// maxEditDist contains the maximum edit distance between a name passed to
// Get*MBIDFromURL and a name in the MusicBrainz database.
const maxEditDist = 2

// GetArtistMBIDFromURL attempts to find the MBID of the artist corresponding to url.
// name should contain the artist name as seen online.
func GetArtistMBIDFromURL(ctx context.Context, db *mbdb.DB, url, name string) string {
	artists, err := db.GetArtistsFromURL(ctx, url)
	if err != nil {
		log.Printf("Failed getting artist MBID from %s: %v", url, err)
		return ""
	}
	return getBestMBID(artists, name)
}

// GetLabelBIDFromURL attempts to find the MBID of the label corresponding to url.
// name should contain the label name as seen online.
func GetLabelMBIDFromURL(ctx context.Context, db *mbdb.DB, url, name string) string {
	labels, err := db.GetLabelsFromURL(ctx, url)
	if err != nil {
		log.Printf("Failed getting label MBID from %s: %v", url, err)
		return ""
	}
	return getBestMBID(labels, name)
}

// getBestMBID returns the MBID of the entity in infos with a name closest to the supplied name
// (as determined by Levenshtein edit distance after lowercasing). If maxEditDist is non-negative,
// it specifies a maximum allowed edit distance. If no matching entity is found, an empty string
// is returned.
func getBestMBID(infos []mbdb.EntityInfo, name string) string {
	if len(infos) == 0 {
		return ""
	}

	// Special case: if only one entity was found in the database and we don't have a name to
	// compare it against, return it. This handles cases like https://syssistersounds.bandcamp.com/,
	// which seems to be an artist page representing a label with releases by many different artists
	// but no link back to the label page.
	if name == "" && len(infos) == 1 {
		log.Printf("Using %v (only entity)", infos[0].MBID)
		return infos[0].MBID
	}

	// Lowercase, de-accent, and decompose strings before comparing them.
	norm := func(s string) string { return strutil.Normalize(strings.ToLower(s)) }
	normName := norm(name)

	var bestDist int
	var bestMBID string
	for _, info := range infos {
		infoName := norm(info.Name)
		dist := strutil.Levenshtein(normName, infoName).Dist()
		if dist > maxEditDist {
			continue
		}
		// Require matches of very short strings (e.g. "A" vs. "B") to be exact.
		if dist > 0 && len(name) <= dist {
			continue
		}
		if bestMBID == "" || dist < bestDist {
			bestDist = dist
			bestMBID = info.MBID
		}
	}
	if bestMBID != "" {
		log.Printf("Using %v for %q (edit distance %d)", bestMBID, name, bestDist)
	}
	return bestMBID
}
