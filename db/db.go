// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package db contains functionality related to the MusicBrainz database.
package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// artistIDs maps from string MBID to int32 database ID.
var artistIDs sync.Map

var artistIDForTest int32

// SetArtistIDForTest hardcodes an ID for GetArtistID to return.
func SetArtistIDForTest(id int32) {
	artistIDForTest = id
}

// GetArtistID queries MusicBrainz over HTTP and returns the database ID (artist.id)
// corresponding to the artist entity with the specified MBID (artist.gid).
func GetArtistID(ctx context.Context, mbid string) (int32, error) {
	if artistIDForTest != 0 {
		return artistIDForTest, nil
	}

	id, ok := artistIDs.Load(mbid)
	if ok {
		return id.(int32), nil
	}

	url := "https://musicbrainz.org/ws/js/entity/" + mbid
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("server returned %v: %v", resp.StatusCode, resp.Status)
	}
	var data struct {
		ID int32 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	if data.ID == 0 {
		return 0, errors.New("server didn't return ID")
	}
	artistIDs.Store(mbid, data.ID)
	return data.ID, nil
}
