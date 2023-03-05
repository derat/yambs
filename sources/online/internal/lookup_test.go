// Copyright 2023 Daniel Erat.
// All rights reserved.

package internal

import (
	"testing"

	"github.com/derat/yambs/mbdb"
)

func TestGetBestMBID(t *testing.T) {
	mkInfos := mbdb.MakeEntityInfosForTest
	for _, tc := range []struct {
		infos      []mbdb.EntityInfo
		name, want string
	}{
		{nil, "Artist", ""},                                             // no entities
		{mkInfos("123", "Artist"), "Artist", "123"},                     // exact match
		{mkInfos("123", "Artist"), "aRTIST", "123"},                     // case-insensitive
		{mkInfos("123", "aRTIST"), "Artist", "123"},                     // case-insensitive (other direction)
		{mkInfos("123", "Artist"), "ÅřŤíşŦ", "123"},                     // ignore diacritics
		{mkInfos("123", "Artist"), "Artist.", "123"},                    // close match
		{mkInfos("123", "Artist"), "Artista.", "123"},                   // exactly maxEditDist (2)
		{mkInfos("123", "Artist"), ".Artista.", ""},                     // too different
		{mkInfos("123", "Artist", "456", "Someone"), "Artist", "123"},   // matches first
		{mkInfos("123", "Artist", "456", "Someone"), "Someone", "456"},  // matches second
		{mkInfos("123", "Artist", "456", "Someone"), "Some_one", "456"}, // closest to second
		{mkInfos("123", "Artist", "456", "Someone"), "Other", ""},       // matches neither
		{mkInfos("123", "A"), "C", ""},                                  // close match but too short
		{mkInfos("123", "A", "456", "B"), "C", ""},                      // close match but too short
		{mkInfos("123", "A", "456", "B"), "A", "123"},                   // allow short name if exact match
		{mkInfos("123", "Artist"), "", "123"},                           // special case: empty name and single entity
		{mkInfos("123", "Artist", "456", "Someone"), "", ""},            // no special case when multiple entities
	} {
		if got := getBestMBID(tc.infos, tc.name); got != tc.want {
			t.Errorf("getBestMBID(%v, %q) = %q; want %q", tc.infos, tc.name, got, tc.want)
		}
	}
}
