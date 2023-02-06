// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package db contains functionality related to the MusicBrainz database.
package db

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/derat/yambs/cache"
	"golang.org/x/time/rate"
)

const (
	// https://musicbrainz.org/doc/MusicBrainz_API/Rate_Limiting
	maxQPS         = 1
	rateBucketSize = 1
	userAgentFmt   = "yambs/%s ( https://github.com/derat/yambs )"

	// TODO: Should cache entries also expire after a certain amount of time?
	cacheSize     = 256         // size for various caches
	cacheMissTime = time.Minute // TTL for negative caches

	defaultServer = "musicbrainz.org"
)

// entityType is an entity type sent to the MusicBrainz API.
type entityType string

const (
	artistType entityType = "artist"
	labelType  entityType = "label"
)

// DB queries the MusicBrainz database using its API.
// See https://musicbrainz.org/doc/MusicBrainz_API.
type DB struct {
	databaseIDs *cache.LRU                // string MBID to int32 database ID
	urlMBIDs    map[entityType]*cache.LRU // string URL to string MBID
	urlMiss     map[entityType]*cache.LRU // string URL to time.Time of negative lookup

	limiter         *rate.Limiter // rate-limits network requests
	disallowQueries bool          // don't allow network traffic
	server          string        // server hostname
	version         string        // included in User-Agent header
}

// NewDB returns a new DB object.
func NewDB(opts ...Option) *DB {
	db := DB{
		databaseIDs: cache.NewLRU(cacheSize),
		urlMBIDs: map[entityType]*cache.LRU{
			artistType: cache.NewLRU(cacheSize),
			labelType:  cache.NewLRU(cacheSize),
		},
		urlMiss: map[entityType]*cache.LRU{
			artistType: cache.NewLRU(cacheSize),
			labelType:  cache.NewLRU(cacheSize),
		},
		limiter: rate.NewLimiter(maxQPS, rateBucketSize),
		server:  defaultServer,
	}
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

// Server returns an Option that configure DB to make calls to the specified
// hostname, e.g. "musicbrains.org" or "test.musicbrainz.org".
func Server(s string) Option { return func(db *DB) { db.server = s } }

// Version returns an Option that sets the application version for the
// User-Agent header.
func Version(v string) Option { return func(db *DB) { db.version = v } }

// GetDatabaseID returns the database ID (e.g. artist.id) corresponding to
// the entity with the specified MBID (e.g. artist.gid).
func (db *DB) GetDatabaseID(ctx context.Context, mbid string) (int32, error) {
	if !IsMBID(mbid) {
		return 0, errors.New("malformed MBID")
	}

	if id, ok := db.databaseIDs.Get(mbid); ok {
		return id.(int32), nil
	}

	// Actually query the database. The /ws/js endpoints apparently exist
	// for field completion rather than being part of the API (/ws/2).
	// See https://wiki.musicbrainz.org/Development/Search_Architecture.
	log.Print("Requesting database ID for ", mbid)
	r, err := db.doQuery(ctx, "https://"+db.server+"/ws/js/entity/"+mbid)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	var data struct {
		ID int32 `json:"id"`
	}
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return 0, err
	} else if data.ID == 0 {
		return 0, errors.New("server didn't return ID")
	}
	log.Print("Got database ID ", data.ID)
	db.databaseIDs.Set(mbid, data.ID)
	return data.ID, nil
}

// GetArtistMBIDFromURL returns the MBID of the artist related to linkURL.
// If no artist is related to the URL, an empty string is returned.
func (db *DB) GetArtistMBIDFromURL(ctx context.Context, linkURL string) (string, error) {
	return db.getMBIDFromURL(ctx, linkURL, artistType)
}

// GetLabelMBIDFromURL returns the MBID of the label related to linkURL.
// If no label is related to the URL, an empty string is returned.
func (db *DB) GetLabelMBIDFromURL(ctx context.Context, linkURL string) (string, error) {
	return db.getMBIDFromURL(ctx, linkURL, labelType)
}

// getMBIDFromURL returns the MBID of the specified entity type related to linkURL.
// An empty string is returned if no relations are found.
func (db *DB) getMBIDFromURL(ctx context.Context, linkURL string, entity entityType) (string, error) {
	// Check the cache first.
	cache := db.urlMBIDs[entity]
	if mbid, ok := cache.Get(linkURL); ok {
		return mbid.(string), nil
	}

	// If we're being called from a test, just pretend like the URL is missing.
	if db.disallowQueries {
		return "", nil
	}

	// Give up if we already checked recently.
	missCache := db.urlMiss[entity]
	if v, ok := missCache.Get(linkURL); ok && time.Now().Sub(v.(time.Time)) <= cacheMissTime {
		return "", nil
	}

	log.Printf("Requesting %v MBID for %v", entity, linkURL)
	reqURL := fmt.Sprintf("https://%s/ws/2/url?resource=%s&inc=%s-rels",
		db.server, url.QueryEscape(linkURL), entity)
	r, err := db.doQuery(ctx, reqURL)
	if err == notFoundError {
		missCache.Set(linkURL, time.Now())
		return "", nil
	} else if err != nil {
		return "", err
	}
	defer r.Close()

	// Parse an XML response like the following:
	//
	//  <?xml version="1.0" encoding="UTF-8"?>
	//  <metadata xmlns="http://musicbrainz.org/ns/mmd-2.0#">
	//    <url id="010fc5d6-2ef6-4852-9075-61184fdc972a">
	//  	<resource>https://pillarsinthesky.bandcamp.com/</resource>
	//  	<relation-list target-type="artist">
	//  	  <relation type="bandcamp" type-id="c550166e-0548-4a18-b1d4-e2ae423a3e88">
	//  		<target>7ba8b326-34ba-472b-b710-b01dc1f14f94</target>
	//  		<direction>backward</direction>
	//  		<artist id="7ba8b326-34ba-472b-b710-b01dc1f14f94" type="Person" type-id="b6e035f4-3ce9-331c-97df-83397230b0df">
	//  		  <name>Pillars in the Sky</name>
	//  		  <sort-name>Pillars in the Sky</sort-name>
	//  		</artist>
	//  	  </relation>
	//  	</relation-list>
	//    </url>
	//  </metadata>
	var md struct {
		XMLName       xml.Name `xml:"metadata"`
		RelationLists []struct {
			TargetType string   `xml:"target-type,attr"`
			Relations  []string `xml:"relation>target"`
		} `xml:"url>relation-list"`
	}
	if err := xml.NewDecoder(r).Decode(&md); err != nil {
		return "", err
	}
	for _, list := range md.RelationLists {
		if entityType(list.TargetType) != entity {
			continue
		}
		// TODO: Figure out a better way to handle multiple relations.
		if nr := len(list.Relations); nr != 1 {
			return "", fmt.Errorf("got %d relations for URL", nr)
		}
		mbid := list.Relations[0]
		log.Print("Got MBID ", mbid)
		cache.Set(linkURL, mbid)
		return mbid, nil
	}
	return "", nil
}

// notFoundError is returned by doQuery if a 404 error was received.
var notFoundError = errors.New("not found")

// doQuery sends a GET request for url and returns the response body.
// The caller should close the body if error is non-nil.
func (db *DB) doQuery(ctx context.Context, url string) (io.ReadCloser, error) {
	if db.disallowQueries {
		return nil, errors.New("querying not allowed")
	}

	// Wait until we can perform a query.
	// TODO: We could be smarter here and bail out early if someone else
	// successfully fetches the same thing while we're waiting.
	if err := db.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	log.Print("Sending GET request for ", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf(userAgentFmt, db.version))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		if resp.StatusCode == 404 {
			return nil, notFoundError
		}
		return nil, fmt.Errorf("server returned %v: %v", resp.StatusCode, resp.Status)
	}
	return resp.Body, nil
}

// SetDatabaseIDForTest hardcodes an ID for GetDatabaseID to return.
func (db *DB) SetDatabaseIDForTest(mbid string, id int32) {
	db.databaseIDs.Set(mbid, id)
}

// SetArtistMBIDFromURLForTest hardcodes an MBID for GetArtistMBIDFromURL to return.
func (db *DB) SetArtistMBIDFromURLForTest(url, mbid string) {
	db.urlMBIDs[artistType].Set(url, mbid)
}

// SetLabelMBIDFromURLForTest hardcodes an MBID for GetLabelMBIDFromURL to return.
func (db *DB) SetLabelMBIDFromURLForTest(url, mbid string) {
	db.urlMBIDs[labelType].Set(url, mbid)
}

// mbidRegexp matches a MusicBrainz ID (i.e. a UUID).
var mbidRegexp = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsMBID returns true if mbid looks like a correctly-formatted MBID (i.e. a UUID).
// Note that this method does not check that the MBID is actually assigned to anything.
func IsMBID(mbid string) bool { return mbidRegexp.MatchString(mbid) }
