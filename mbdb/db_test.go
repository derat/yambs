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
		singleURL = "https://listen.tidal.com/artist/3634161"
		// https://musicbrainz.org/ws/2/url?resource=https%3A%2F%2Flisten.tidal.com%2Fartist%2F3634161&inc=artist-rels
		// fetched on 2023-02-06.
		singleData = `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://musicbrainz.org/ns/mmd-2.0#"><url id="a5d16dcf-6b6f-4480-9791-a43272e984b8"><resource>https://listen.tidal.com/artist/3634161</resource><relation-list target-type="artist"><relation type="streaming" type-id="63cc5d1f-f096-4c94-a43f-ecb32ea94161"><target>b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d</target><direction>backward</direction><artist id="b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b"><name>The Beatles</name><sort-name>Beatles, The</sort-name></artist></relation></relation-list></url></metadata>`
		singleName = "The Beatles"
		singleMBID = "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"

		multiURL = "https://mpsc.bandcamp.com"
		// https://musicbrainz.org/ws/2/url?resource=https%3A%2F%2Fmpsc.bandcamp.com%2F&inc=artist-rels
		// fetched on 2023-02-21.
		multiData = `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://musicbrainz.org/ns/mmd-2.0#"><url id="726abe3f-03e8-479a-8fff-237be75b78e9"><resource>https://mpsc.bandcamp.com/</resource><relation-list target-type="artist"><relation type="bandcamp" type-id="c550166e-0548-4a18-b1d4-e2ae423a3e88"><target>4e1a1a2c-32c5-4013-9264-61b92e06b7d4</target><direction>backward</direction><artist id="4e1a1a2c-32c5-4013-9264-61b92e06b7d4" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b"><name>Hanz Mambo &amp; his Cigarettes</name><sort-name>Hanz Mambo &amp; his Cigarettes</sort-name><disambiguation>aka Misha Panfilov Sound Combo</disambiguation></artist></relation><relation type="bandcamp" type-id="c550166e-0548-4a18-b1d4-e2ae423a3e88"><target>c6d215c4-c718-4bb6-a54a-4c1eee8bc068</target><direction>backward</direction><artist id="c6d215c4-c718-4bb6-a54a-4c1eee8bc068" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b"><name>Misha Panfilov Sound Combo</name><sort-name>Misha Panfilov Sound Combo</sort-name></artist></relation><relation type="bandcamp" type-id="c550166e-0548-4a18-b1d4-e2ae423a3e88"><target>eef9b345-2daa-4209-b595-a3e290335f64</target><direction>backward</direction><artist id="eef9b345-2daa-4209-b595-a3e290335f64" type="Person" type-id="b6e035f4-3ce9-331c-97df-83397230b0df"><name>Misha Panfilov</name><sort-name>Panfilov, Misha</sort-name></artist></relation></relation-list></url></metadata>`
		multiName1 = "Hanz Mambo & his Cigarettes"
		multiName2 = "Misha Panfilov Sound Combo"
		multiName3 = "Misha Panfilov"
		multiMBID1 = "4e1a1a2c-32c5-4013-9264-61b92e06b7d4"
		multiMBID2 = "c6d215c4-c718-4bb6-a54a-4c1eee8bc068"
		multiMBID3 = "eef9b345-2daa-4209-b595-a3e290335f64"

		missingURL  = "http://example.org/bogus-url"
		missingName = "foo"
		missingData = `<?xml version="1.0" encoding="UTF-8"?>
<error><text>Not Found</text><text>For usage, please see: https://musicbrainz.org/development/mmd</text></error>`
	)

	singlePath := "/ws/2/url?resource=" + url.QueryEscape(singleURL) + "&inc=artist-rels"
	multiPath := "/ws/2/url?resource=" + url.QueryEscape(multiURL) + "&inc=artist-rels"
	missingPath := "/ws/2/url?resource=" + url.QueryEscape(missingURL) + "&inc=artist-rels"

	reqs := make(map[string]int) // path with query to request count
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path + "?" + r.URL.RawQuery
		switch p {
		case singlePath:
			io.WriteString(w, singleData)
		case multiPath:
			io.WriteString(w, multiData)
		case missingPath:
			http.Error(w, missingData, http.StatusNotFound)
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
	for _, tc := range []struct{ url, name, want string }{
		{singleURL, singleName, singleMBID},
		{singleURL, singleName, singleMBID},
		{multiURL, multiName1, multiMBID1},
		{multiURL, multiName2, multiMBID2},
		{multiURL, multiName3, multiMBID3},
		{multiURL, multiName1, multiMBID1},
		{multiURL, multiName2, multiMBID2},
		{multiURL, multiName3, multiMBID3},
		{missingURL, missingName, ""},
		{missingURL, missingName, ""},
	} {
		if got, err := db.GetArtistMBIDFromURL(ctx, tc.url, tc.name); err != nil {
			t.Errorf("GetArtistMBIDFromURL(ctx, %q, %q) failed: %v", tc.url, tc.name, err)
		} else if got != tc.want {
			t.Errorf("GetArtistMBIDFromURL(ctx, %q, %q) = %q; want %q", tc.url, tc.name, got, tc.want)
		}
	}

	// Check that positive and negative results were cached.
	if want := map[string]int{
		singlePath:  1,
		multiPath:   1,
		missingPath: 1,
	}; !reflect.DeepEqual(reqs, want) {
		t.Errorf("Got %v; want %v", reqs, want)
	}

	// Verify that cached misses expire.
	now = now.Add(cacheMissTime + time.Second)
	if got, err := db.GetArtistMBIDFromURL(ctx, missingURL, missingName); err != nil {
		t.Errorf("GetArtistMBIDFromURL(ctx, %q, %q) failed: %v", missingURL, missingName, err)
	} else if got != "" {
		t.Errorf("GetArtistMBIDFromURL(ctx, %q, %q) = %q; want %q", missingURL, missingName, got, "")
	}
	if cnt := reqs[missingPath]; cnt != 2 {
		t.Errorf("Got %d request(s) for %q; want 2", cnt, missingPath)
	}
}
