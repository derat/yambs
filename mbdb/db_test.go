// Copyright 2023 Daniel Erat.
// All rights reserved.

package mbdb

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestDB_GetDatabaseID(t *testing.T) {
	const (
		mbid = "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"
		path = "/ws/js/entity/" + mbid
		// https://musicbrainz.org/ws/js/entity/b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d
		// fetched on 2022-02-06.
		data = `{"area":null,"editsPending":false,"name":"The Beatles","comment":"","last_updated":"2022-11-20T08:00:36Z","gid":"b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d","id":303,"ended":true,"ipi_codes":[],"end_area_id":null,"entityType":"artist","sort_name":"Beatles, The","begin_date":{"year":1960,"month":null,"day":null},"begin_area_id":3924,"isni_codes":[],"end_date":{"month":4,"day":10,"year":1970},"typeID":2,"gender_id":null}`
		id   = 303
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Fatalf("Got request for %q; want %q", r.URL.Path, path)
		}
		io.WriteString(w, data)
	}))
	defer srv.Close()

	db := NewDB(ServerURL(srv.URL))
	if got, err := db.GetDatabaseID(context.Background(), mbid); err != nil {
		t.Fatalf("GetDatabaseID(ctx, %q) failed: %v", mbid, err)
	} else if got != id {
		t.Fatalf("GetDatabaseID(ctx, %q) = %d; want %d", mbid, got, id)
	}
}

func TestDB_GetArtistMBIDFromURL(t *testing.T) {
	const (
		goodURL = "https://listen.tidal.com/artist/3634161"
		// https://musicbrainz.org/ws/2/url?resource=https%3A%2F%2Flisten.tidal.com%2Fartist%2F3634161&inc=artist-rels
		// fetched on 2022-02-06.
		goodData = `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://musicbrainz.org/ns/mmd-2.0#"><url id="a5d16dcf-6b6f-4480-9791-a43272e984b8"><resource>https://listen.tidal.com/artist/3634161</resource><relation-list target-type="artist"><relation type="streaming" type-id="63cc5d1f-f096-4c94-a43f-ecb32ea94161"><target>b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d</target><direction>backward</direction><artist id="b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b"><name>The Beatles</name><sort-name>Beatles, The</sort-name></artist></relation></relation-list></url></metadata>`
		goodMBID = "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"

		badURL  = "http://example.org/bogus-url"
		badData = `<?xml version="1.0" encoding="UTF-8"?>
<error><text>Not Found</text><text>For usage, please see: https://musicbrainz.org/development/mmd</text></error>`
	)

	goodPath := "/ws/2/url?resource=" + url.QueryEscape(goodURL) + "&inc=artist-rels"
	badPath := "/ws/2/url?resource=" + url.QueryEscape(badURL) + "&inc=artist-rels"

	reqs := make(map[string]int) // path with query to request count
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path + "?" + r.URL.RawQuery
		switch p {
		case goodPath:
			io.WriteString(w, goodData)
		case badPath:
			http.Error(w, badData, http.StatusNotFound)
		default:
			t.Fatalf("Got request for %q", p)
		}
		reqs[p]++
	}))
	defer srv.Close()

	ctx := context.Background()
	now := time.Unix(0, 0)
	nowFunc := func() time.Time { return now }
	db := NewDB(ServerURL(srv.URL), MaxQPS(999), NowFunc(nowFunc))
	for _, tc := range []struct{ url, want string }{
		{goodURL, goodMBID},
		{goodURL, goodMBID},
		{badURL, ""},
		{badURL, ""},
	} {
		if got, err := db.GetArtistMBIDFromURL(ctx, tc.url); err != nil {
			t.Errorf("GetArtistMBIDFromURL(ctx, %q) failed: %v", tc.url, err)
		} else if got != tc.want {
			t.Errorf("GetArtistMBIDFromURL(ctx, %q) = %q; want %q", tc.url, got, tc.want)
		}
	}

	// Check that positive and negative results were cached.
	if want := map[string]int{goodPath: 1, badPath: 1}; !reflect.DeepEqual(reqs, want) {
		t.Errorf("Got %v; want %v", reqs, want)
	}

	// Verify that cached misses expire.
	now = now.Add(cacheMissTime + time.Second)
	if got, err := db.GetArtistMBIDFromURL(ctx, badURL); err != nil {
		t.Errorf("GetArtistMBIDFromURL(ctx, %q) failed: %v", badURL, err)
	} else if got != "" {
		t.Errorf("GetArtistMBIDFromURL(ctx, %q) = %q; want %q", badURL, got, "")
	}
	if cnt := reqs[badPath]; cnt != 2 {
		t.Errorf("Got %d request(s) for %q; want 2", cnt, badPath)
	}
}
