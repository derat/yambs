// Copyright 2022 Daniel Erat.
// All rights reserved.

package qobuz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/derat/yambs/db"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/web"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

func TestRelease(t *testing.T) {
	ctx := context.Background()
	db := db.NewDB(db.DisallowQueries)
	var pr Provider

	for _, tc := range []struct {
		url string
		rel *seed.Release
		img string
	}{
		{
			url: "https://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102",
			rel: &seed.Release{
				Title:     "A Dave Brubeck Christmas",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				// No release event since it predates Qobuz's launch date.
				Labels:  []seed.ReleaseLabel{{Name: "Telarc"}},
				Artists: []seed.ArtistCredit{{Name: "Dave Brubeck"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						track(`"Homecoming" Jingle Bells`, "00:03:20"),
						track("Santa Claus Is Coming To Town", "00:03:38"),
						track("Joy To The World", "00:02:53"),
						track("Away In A Manger", "00:05:03"),
						track("Winter Wonderland", "00:04:19"),
						track("O Little Town Of Bethlehem", "00:05:34"),
						track("What Child Is This? (Greensleeves)", "00:03:27"),
						track("To Us Is Given", "00:03:32"),
						track("O Tannenbaum", "00:03:35"),
						track("Silent Night", "00:04:53"),
						track("Cantos para Pedir las Posadas", "00:03:59"),
						track("Run, Run, Run To Bethlehem", "00:03:48"),
						track(`"Farewell" Jingle Bells`, "00:02:59"),
						track("The Christmas Song", "00:04:28"),
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://open.qobuz.com/album/0008940834102",
						LinkType: seed.LinkType_StreamingPaid_Release_URL,
					},
				},
			},
			img: "https://static.qobuz.com/images/covers/02/41/0008940834102_max.jpg",
		},
		{
			url: "https://www.qobuz.com/us-en/album/in-rainbows-radiohead/0634904032432",
			rel: &seed.Release{
				Title:     "In Rainbows",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Year: 2007, Month: 10, Day: 10}},
				Labels:    []seed.ReleaseLabel{{Name: "XL Recordings"}},
				Artists:   []seed.ArtistCredit{{Name: "Radiohead"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						track("15 Step", "00:03:57"),
						track("Bodysnatchers", "00:04:02"),
						track("Nude", "00:04:15"),
						track("Weird Fishes/ Arpeggi", "00:05:18"),
						track("All I Need", "00:03:49"),
						track("Faust Arp", "00:02:10"),
						track("Reckoner", "00:04:50"),
						track("House Of Cards", "00:05:28"),
						track("Jigsaw Falling Into Place", "00:04:09"),
						track("Videotape", "00:04:40"),
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://www.qobuz.com/album/in-rainbows-radiohead/0634904032432",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://open.qobuz.com/album/0634904032432",
						LinkType: seed.LinkType_StreamingPaid_Release_URL,
					},
				},
			},
			img: "https://static.qobuz.com/images/covers/32/24/0634904032432_max.jpg",
		},
		{
			url: "https://www.qobuz.com/us-en/album/the-dark-side-of-the-moon-pink-floyd/xggxq5w5dmljb",
			rel: &seed.Release{
				Title:     "The Dark Side of the Moon",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Labels:    []seed.ReleaseLabel{{Name: "Pink Floyd Records"}},
				Artists:   []seed.ArtistCredit{{Name: "Pink Floyd"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						track("Speak to Me", "00:01:04"),
						track("Breathe (In the Air)", "00:02:49"),
						track("On the Run", "00:03:36"),
						track("Time", "00:07:02"),
						track("The Great Gig in the Sky", "00:04:44"),
						track("Money", "00:06:20"),
						track("Us and Them", "00:07:52"),
						track("Any Colour You Like", "00:03:25"),
						track("Brain Damage", "00:03:50"),
						track("Eclipse", "00:02:06"),
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://www.qobuz.com/album/the-dark-side-of-the-moon-pink-floyd/xggxq5w5dmljb",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://open.qobuz.com/album/xggxq5w5dmljb",
						LinkType: seed.LinkType_StreamingPaid_Release_URL,
					},
				},
			},
			img: "https://static.qobuz.com/images/covers/jb/ml/xggxq5w5dmljb_max.jpg",
		},
	} {
		t.Run(tc.url, func(t *testing.T) {
			f, err := os.Open(getFilename(tc.url))
			if err != nil {
				t.Fatal("Failed opening page:", err)
			}
			defer f.Close()

			root, err := html.Parse(f)
			if err != nil {
				t.Fatal("Failed parsing HTML:", err)
			}
			page := &web.Page{Root: root}
			rel, img, err := pr.Release(ctx, page, tc.url, db)
			if err != nil {
				t.Fatal("Failed parsing page:", err)
			}

			if diff := cmp.Diff(tc.rel, rel); diff != "" {
				t.Error("Bad release data:\n" + diff)
			}

			var imgURL string
			if img != nil {
				imgURL = img.URL("" /* srv */)
			}
			if diff := cmp.Diff(tc.img, imgURL); diff != "" {
				t.Error("Bad cover image URL:\n" + diff)
			}
		})
	}
}

// getFilename converts a URL to a file in the testdata directory.
func getFilename(url string) string {
	fn := strings.TrimPrefix(url, "https://")
	fn = strings.ReplaceAll(fn, "/", "_")
	fn += ".html"
	return filepath.Join("testdata", fn)
}

// track constructs a seed.Track from the supplied title and "HH:MM:SS" duration.
func track(title, dur string) seed.Track {
	d, err := parseDuration(dur)
	if err != nil {
		panic(fmt.Sprintf("Bad %q duration %q: %v", title, dur, err))
	}
	return seed.Track{Title: title, Length: d}
}

func TestCleanURL(t *testing.T) {
	var pr Provider
	for _, tc := range []struct {
		in   string
		want string
		ok   bool // if false, error should be returned
	}{
		{"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102",
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"https://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102/?utm_source=foo#bar",
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"http://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102",
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"https://www.qobuz.com/us-en/album/fearless-taylors-version-taylor-swift/r8361te6k0cic",
			"https://www.qobuz.com/album/fearless-taylors-version-taylor-swift/r8361te6k0cic", true},
		{"https://www.qobuz.com/us-en/discover", "", false},
		{"https://example.org/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", "", false},
	} {
		if got, err := pr.CleanURL(tc.in); !tc.ok && err == nil {
			t.Errorf("CleanURL(%q) = %q; wanted error", tc.in, got)
		} else if tc.ok && err != nil {
			t.Errorf("CleanURL(%q) failed: %v", tc.in, err)
		} else if tc.ok && got != tc.want {
			t.Errorf("CleanURL(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
