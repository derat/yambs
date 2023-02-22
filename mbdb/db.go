// Copyright 2022 Daniel Erat.
// All rights reserved.

// Package mbdb contains functionality related to the MusicBrainz database.
package mbdb

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

	defaultServerURL = "https://musicbrainz.org"
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
	urlRels     map[entityType]*cache.LRU // string URL to []urlRel
	urlMiss     map[entityType]*cache.LRU // string URL to time.Time of negative lookup

	limiter         *rate.Limiter    // rate-limits network requests
	disallowQueries bool             // don't allow network traffic
	serverURL       string           // base server URL without trailing slash
	version         string           // included in User-Agent header
	now             func() time.Time // called to get current time
}

// urlRel holds details about an entity related to a URL.
type urlRel struct {
	mbid string
	name string // name of artist/label/etc. in MB
}

// NewDB returns a new DB object.
func NewDB(opts ...Option) *DB {
	db := DB{
		databaseIDs: cache.NewLRU(cacheSize),
		urlRels: map[entityType]*cache.LRU{
			artistType: cache.NewLRU(cacheSize),
			labelType:  cache.NewLRU(cacheSize),
		},
		urlMiss: map[entityType]*cache.LRU{
			artistType: cache.NewLRU(cacheSize),
			labelType:  cache.NewLRU(cacheSize),
		},
		limiter:   rate.NewLimiter(maxQPS, rateBucketSize),
		serverURL: defaultServerURL,
		now:       time.Now,
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

// ServerURL returns an Option that configure DB to make calls to the specified
// base server URL, e.g. "https://musicbrains.org" or "https://test.musicbrainz.org".
func ServerURL(u string) Option { return func(db *DB) { db.serverURL = u } }

// Version returns an Option that sets the application version for the User-Agent header.
func Version(v string) Option { return func(db *DB) { db.version = v } }

// NowFunc injects a function that is called instead of time.Now to get the current time.
func NowFunc(fn func() time.Time) Option { return func(db *DB) { db.now = fn } }

// MaxQPS overrides the default QPS limit for testing.
func MaxQPS(qps int) Option { return func(db *DB) { db.limiter.SetLimit(rate.Limit(qps)) } }

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
	r, err := db.doQuery(ctx, "/ws/js/entity/"+mbid)
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
// The supplied artist name is used to choose an entity if multiple entities are related to the URL.
// If no artist is related to the URL, an empty string is returned.
func (db *DB) GetArtistMBIDFromURL(ctx context.Context, linkURL, name string) (string, error) {
	return db.getMBIDFromURL(ctx, linkURL, artistType, name)
}

// GetLabelMBIDFromURL returns the MBID of the label related to linkURL.
// The supplied label name is used to choose an entity if multiple entities are related to the URL.
// If no label is related to the URL, an empty string is returned.
func (db *DB) GetLabelMBIDFromURL(ctx context.Context, linkURL, name string) (string, error) {
	return db.getMBIDFromURL(ctx, linkURL, labelType, name)
}

// getMBIDFromURL returns the MBID of the specified entity type related to linkURL.
// The supplied name is used to choose an entity if multiple entities are related to the URL.
// An empty string is returned if no relations are found.
func (db *DB) getMBIDFromURL(ctx context.Context, linkURL string, entity entityType, name string) (string, error) {
	rels, err := db.getURLRels(ctx, linkURL, entity)
	if err != nil {
		return "", err
	}
	var mbid string
	var dist int
	for _, rel := range rels {
		// If there are multiple relations, choose the one with the shortest edit distance from the passed-in name.
		if d := levenshtein(name, rel.name).dist(); mbid == "" || d < dist {
			mbid = rel.mbid
			dist = d
		}
	}
	if mbid != "" {
		log.Printf("Using MBID %v for %v (%q)", mbid, linkURL, name)
	}
	return mbid, nil
}

// getURLRels returns entities of the specified type related to linkURL.
func (db *DB) getURLRels(ctx context.Context, linkURL string, entity entityType) ([]urlRel, error) {
	// Check the cache first.
	cache := db.urlRels[entity]
	if rels, ok := cache.Get(linkURL); ok {
		return rels.([]urlRel), nil
	}

	// If we're being called from a test, just pretend like the URL is missing.
	if db.disallowQueries {
		return nil, nil
	}

	// Give up if we already checked recently.
	missCache := db.urlMiss[entity]
	if v, ok := missCache.Get(linkURL); ok && db.now().Sub(v.(time.Time)) <= cacheMissTime {
		return nil, nil
	}

	log.Printf("Requesting %v relations for %v", entity, linkURL)
	path := fmt.Sprintf("/ws/2/url?resource=%s&inc=%s-rels", url.QueryEscape(linkURL), entity)
	r, err := db.doQuery(ctx, path)
	if err == notFoundError {
		missCache.Set(linkURL, db.now())
		return nil, nil
	} else if err != nil {
		return nil, err
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
			TargetType string `xml:"target-type,attr"`
			Relations  []struct {
				Target     string `xml:"target"`
				ArtistName string `xml:"artist>name"`
				LabelName  string `xml:"label>name"`
			} `xml:"relation"`
		} `xml:"url>relation-list"`
	}
	if err := xml.NewDecoder(r).Decode(&md); err != nil {
		return nil, err
	}

	var rels []urlRel
	for _, list := range md.RelationLists {
		if entityType(list.TargetType) != entity {
			continue
		}
		for _, rel := range list.Relations {
			var name string
			switch entity {
			case artistType:
				name = rel.ArtistName
			case labelType:
				name = rel.LabelName
			}
			rels = append(rels, urlRel{mbid: rel.Target, name: name})
		}
	}
	log.Printf("Got %d relation(s) for %v", len(rels), linkURL)
	cache.Set(linkURL, rels)
	return rels, nil
}

// notFoundError is returned by doQuery if a 404 error was received.
var notFoundError = errors.New("not found")

// doQuery sends a GET request for path and returns the response body.
// The caller is responsible for closing the body if the error is non-nil.
func (db *DB) doQuery(ctx context.Context, path string) (io.ReadCloser, error) {
	if db.disallowQueries {
		return nil, errors.New("querying not allowed")
	}

	// Wait until we can perform a query.
	// TODO: We could be smarter here and bail out early if someone else
	// successfully fetches the same thing while we're waiting.
	if err := db.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	u := db.serverURL + path
	log.Print("Sending GET request for ", u)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
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
	db.urlRels[artistType].Set(url, []urlRel{{mbid, ""}})
}

// SetLabelMBIDFromURLForTest hardcodes an MBID for GetLabelMBIDFromURL to return.
func (db *DB) SetLabelMBIDFromURLForTest(url, mbid string) {
	db.urlRels[labelType].Set(url, []urlRel{{mbid, ""}})
}

// mbidRegexp matches a MusicBrainz ID (i.e. a UUID).
var mbidRegexp = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsMBID returns true if mbid looks like a correctly-formatted MBID (i.e. a UUID).
// Note that this method does not check that the MBID is actually assigned to anything.
func IsMBID(mbid string) bool { return mbidRegexp.MatchString(mbid) }

// ShortenContext returns a context derived from ctx with its deadline shortened by t.
// If ctx does not have a deadline, a derived deadline-less context is returned.
// The caller must call the returned cancel function to release resources.
func ShortenContext(ctx context.Context, t time.Duration) (context.Context, context.CancelFunc) {
	if dl, ok := ctx.Deadline(); ok {
		return context.WithDeadline(ctx, dl.Add(-t))
	}
	return context.WithCancel(ctx)
}
