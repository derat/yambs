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
	"github.com/derat/yambs/sources/online/internal"
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
						LinkType: seed.LinkType_Streaming_Release_URL,
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
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.Date{Year: 2007, Month: 10, Day: 10}}},
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
						LinkType: seed.LinkType_Streaming_Release_URL,
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
						LinkType: seed.LinkType_Streaming_Release_URL,
					},
				},
			},
			img: "https://static.qobuz.com/images/covers/jb/ml/xggxq5w5dmljb_max.jpg",
		},
		{
			// This album contains per-track artist credits.
			url: "https://www.qobuz.com/us-en/album/waynes-world-various-artists/0093624963714",
			rel: &seed.Release{
				// The extra space here is present throughout the page.
				Title:     "Wayne's World  (Music From The Motion Picture)",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Labels:    []seed.ReleaseLabel{{Name: "Reprise"}},
				Artists:   []seed.ArtistCredit{{Name: "Various Artists"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						trackArtist("Bohemian Rhapsody", "Queen", "00:05:57"),
						trackArtist("Hot and Bothered (Album Version)", "Cinderella", "00:04:16"),
						trackArtist("Rock Candy (Album Version)", "Bulletboys", "00:05:04"),
						trackArtist("Dream Weaver (Wayne's World Version) (Album Version)", "Gary Wright", "00:04:25"),
						trackArtist("Sikamikanico (Album Version)", "Red Hot Chili Peppers", "00:03:25"),
						trackArtist("Time Machine (Wayne's World Soundtrack Version) [2000 Remaster] (Album Version)", "Black Sabbath", "00:04:19"),
						trackArtist("Wayne's World Theme (Extended Version)", "Wayne And Garth", "00:05:14"),
						trackArtist("Ballroom Blitz (Album Version)", "Tia Carrere", "00:03:30"),
						trackArtist("Foxey Lady (Album Version)", "Jimi Hendrix", "00:03:19"),
						trackArtist("Feed My Frankenstein (Album Version)", "Alice Cooper", "00:04:46"),
						trackArtist("Ride with Yourself", "Rhino Bucket", "00:03:15"),
						trackArtist("Loving Your Lovin' (Album Version)", "Eric Clapton", "00:03:54"),
						trackArtist("Why You Wanna Break My Heart (Album Version)", "Tia Carrere", "00:03:32"),
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://www.qobuz.com/album/waynes-world-various-artists/0093624963714",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://open.qobuz.com/album/0093624963714",
						LinkType: seed.LinkType_Streaming_Release_URL,
					},
				},
			},
			img: "https://static.qobuz.com/images/covers/14/37/0093624963714_max.jpg",
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
			cfg := internal.Config{DisallowNetwork: true}
			rel, img, err := pr.Release(ctx, page, tc.url, db, &cfg)
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

// trackArtist constructs a seed.Track from the supplied title, artist name, and "HH:MM:SS" duration.
func trackArtist(title, artist, dur string) seed.Track {
	tr := track(title, dur)
	tr.Artists = []seed.ArtistCredit{{Name: artist}}
	return tr
}

func TestCleanURL(t *testing.T) {
	for _, tc := range []struct {
		orig         string
		removeLocale bool
		want         string
		ok           bool // if false, error should be returned
	}{
		{"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true,
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", false,
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"http://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true,
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"http://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", false,
			"https://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"https://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102/?utm_source=foo#bar", true,
			"https://www.qobuz.com/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"https://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102/?utm_source=foo#bar", false,
			"https://www.qobuz.com/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", true},
		{"https://www.qobuz.com/mx-es/album/fearless-taylors-version-taylor-swift/r8361te6k0cic", true,
			"https://www.qobuz.com/album/fearless-taylors-version-taylor-swift/r8361te6k0cic", true},
		{"https://www.qobuz.com/mx-es/album/fearless-taylors-version-taylor-swift/r8361te6k0cic", false,
			"https://www.qobuz.com/mx-es/album/fearless-taylors-version-taylor-swift/r8361te6k0cic", true},
		{"https://www.qobuz.com/us-en/discover", false, "", false},
		{"https://example.org/us-en/album/a-dave-brubeck-christmas-dave-brubeck/0008940834102", false, "", false},
	} {
		if got, err := cleanURL(tc.orig, tc.removeLocale); !tc.ok && err == nil {
			t.Errorf("cleanURL(%q, %v) = %q; wanted error", tc.orig, tc.removeLocale, got)
		} else if tc.ok && err != nil {
			t.Errorf("cleanURL(%q, %v) failed: %v", tc.orig, tc.removeLocale, err)
		} else if tc.ok && got != tc.want {
			t.Errorf("cleanURL(%q, %v) = %q; want %q", tc.orig, tc.removeLocale, got, tc.want)
		}
	}
}
