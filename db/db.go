// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package db contains functionality related to the MusicBrainz database.
package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

const (
	// https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting
	maxQPS         = 1
	rateBucketSize = 1
	userAgentFmt   = "yambs/%s ( https://github.com/derat/yambs )"
)

// DB queries the MusicBrainz database.
type DB struct {
	databaseIDs     sync.Map      // string MBID to int32 database ID
	limiter         *rate.Limiter // rate-limits network requests
	disallowQueries bool          // don't allow network traffic
	version         string        // included in User-Agent header
}

// NewDB returns a new DB object.
func NewDB(opts ...Option) *DB {
	db := DB{limiter: rate.NewLimiter(maxQPS, rateBucketSize)}
	for _, o := range opts {
		o(&db)
	}
	return &db
}

// Option can be passed to NewDB to configure the database.
type Option func(db *DB)

// DisallowQueries is an Option that configures DB to report an error
// when it would need to perform a query over the network.
var DisallowQueries = func(db *DB) { db.disallowQueries = true }

// Version returns an Option that sets the application version for the
// User-Agent header.
func Version(v string) Option { return func(db *DB) { db.version = v } }

// GetDatabaseID returns the database ID (e.g. artist.id) corresponding to
// the entity with the specified MBID (e.g. artist.gid).
func (db *DB) GetDatabaseID(ctx context.Context, mbid string) (int32, error) {
	// TODO: Validate MBID format?
	if id, ok := db.databaseIDs.Load(mbid); ok {
		return id.(int32), nil
	}

	if db.disallowQueries {
		return 0, errors.New("querying not allowed")
	}

	// Actually query the database.
	var data struct {
		ID int32 `json:"id"`
	}
	log.Printf("Requesting database ID for %q", mbid)
	if err := db.doQuery(ctx, "https://musicbrainz.org/ws/js/entity/"+mbid, &data); err != nil {
		return 0, err
	}
	if data.ID == 0 {
		return 0, errors.New("server didn't return ID")
	}
	db.databaseIDs.Store(mbid, data.ID)
	return data.ID, nil
}

// doQuery sends a GET request for url and JSON-unmarshals the response into dst.
func (db *DB) doQuery(ctx context.Context, url string, dst any) error {
	// Wait until we can perform a query.
	// TODO: We could be smarter here and bail out early if someone else
	// successfully fetches the same thing while we're waiting.
	if err := db.limiter.Wait(ctx); err != nil {
		return err
	}

	log.Printf("Sending GET request for %v", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf(userAgentFmt, db.version))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned %v: %v", resp.StatusCode, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

// SetDatabaseIDForTest hardcodes an ID for GetDatabaseID to return.
func (db *DB) SetDatabaseIDForTest(mbid string, id int32) {
	db.databaseIDs.Store(mbid, id)
}
