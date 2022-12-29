// Copyright 2022 Daniel Erat.
// All rights reserved.

package bandcamp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// Add canned MBIDs for artists and label lookups.
	for url, mbid := range map[string]string{
		"https://louiezong.bandcamp.com/":       "0e2c603f-fd71-4ab6-af96-92c3e936586d",
		"https://louiscole.bandcamp.com/":       "525ef747-abf6-423c-98b4-cd49c0c07927",
		"https://pillarsinthesky.bandcamp.com/": "7ba8b326-34ba-472b-b710-b01dc1f14f94",
		"https://thelovelymoon.bandcamp.com/":   "f34ae170-055d-46bc-9208-a750a646481b",
		// "https://volaband.bandcamp.com/album/live-from-the-pool" omitted
	} {
		db.SetArtistMBIDFromURLForTest(url, mbid)
	}
	for url, mbid := range map[string]string{
		"https://brainfeeder.bandcamp.com/": "20b3d6f9-9086-48d9-802f-5f808456a0ef",
		// "https://mascotlabelgroup.bandcamp.com/" omitted
	} {
		db.SetLabelMBIDFromURLForTest(url, mbid)
	}

	for _, tc := range []struct {
		url string
		rel *seed.Release
		img string
	}{
		{
			// This album has a hidden track, which should be included in the tracklist.
			// It also shouldn't have a stream-for-free link since the hidden track can't be streamed.
			url: "https://louiezong.bandcamp.com/album/cartoon-funk",
			rel: &seed.Release{
				Title:     "Cartoon Funk",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Year: 2022, Month: 6, Day: 17, Country: "XW"}},
				Artists: []seed.ArtistCredit{{
					MBID: "0e2c603f-fd71-4ab6-af96-92c3e936586d",
					Name: "Louie Zong",
				}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "A Worm Welcome!", Length: sec(126.224)},
						{Title: "Anxiety Groove", Length: sec(137.928)},
						{Title: "Coelacanth", Length: sec(173.402)},
						{Title: "Brainfoghorn", Length: sec(120.078)},
						{Title: "Sunset Skating", Length: sec(104.019)},
						{Title: "Evolution of the Eye", Length: sec(132.928)},
						{Title: "Jar of Pickles", Length: sec(111.16)},
						{Title: "I Outsolve You", Length: sec(108.172)},
						{Title: "Rough Edges", Length: sec(126.221)},
						{Title: "Signal/Noise (feat. Turner Perez)", Length: sec(137.027)},
						{Title: "Furniture Hellscape", Length: sec(106.083)},
						{Title: "Rooftop Cats", Length: sec(98.3392)},
						{Title: "Spring Cleaning", Length: sec(139.693)},
						{Title: "My Father's Sacred Sword", Length: sec(120.047)},
						{Title: "Virtual Home", Length: sec(93.4912)},
						{Title: "Cartoon Mines", Length: sec(208.526)},
						{Title: "Only A Toad", Length: sec(119.907)},
						{Title: "Worm Cowboy (See Ya!)", Length: sec(63.6559)},
						{Title: "[unknown]"}, // hidden track
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://louiezong.bandcamp.com/album/cartoon-funk",
						LinkType: seed.LinkType_DownloadForFree_Release_URL,
					},
					{
						URL:      "https://louiezong.bandcamp.com/album/cartoon-funk",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
				},
			},
			img: "https://f4.bcbits.com/img/a1689585732_0.jpg",
		},
		{
			// This is an album URL, but it has a single track that matches the album title
			// so it should be treated as a single rather than an album.
			url: "https://louiscole.bandcamp.com/album/let-it-happen",
			rel: &seed.Release{
				Title:     "Let it Happen",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
				Barcode:   "5054429157154",
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Year: 2022, Month: 8, Day: 2, Country: "XW"}},
				Labels:    []seed.ReleaseLabel{{Name: "Brainfeeder"}},
				Artists: []seed.ArtistCredit{{
					MBID: "525ef747-abf6-423c-98b4-cd49c0c07927",
					Name: "Louis cole",
				}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Let it Happen", Length: sec(403)},
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://louiscole.bandcamp.com/album/let-it-happen",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://louiscole.bandcamp.com/album/let-it-happen",
						LinkType: seed.LinkType_StreamingMusic_Release_URL,
					},
				},
			},
			img: "https://f4.bcbits.com/img/a3000320182_0.jpg",
		},
		{
			// This is a non-album track URL, which should be treated as a single.
			url: "https://pillarsinthesky.bandcamp.com/track/arcanum",
			rel: &seed.Release{
				Title:     "Arcanum",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Year: 2015, Month: 5, Day: 3, Country: "XW"}},
				Artists: []seed.ArtistCredit{{
					MBID: "7ba8b326-34ba-472b-b710-b01dc1f14f94",
					Name: "Pillars In The Sky",
				}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Arcanum", Length: sec(341.294)},
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://pillarsinthesky.bandcamp.com/track/arcanum",
						LinkType: seed.LinkType_DownloadForFree_Release_URL,
					},
					{
						URL:      "https://pillarsinthesky.bandcamp.com/track/arcanum",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://pillarsinthesky.bandcamp.com/track/arcanum",
						LinkType: seed.LinkType_StreamingMusic_Release_URL,
					},
				},
			},
			img: "https://f4.bcbits.com/img/a2320496643_0.jpg",
		},
		{
			// This album has a Creative Commons license.
			url: "https://thelovelymoon.bandcamp.com/album/echoes-of-memories",
			rel: &seed.Release{
				Title:     "Echoes of Memories",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Year: 2022, Month: 10, Day: 22, Country: "XW"}},
				Artists: []seed.ArtistCredit{{
					MBID: "f34ae170-055d-46bc-9208-a750a646481b",
					Name: "The Lovely Moon",
				}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Echoes of Memories", Length: sec(2295.17)},
						{Title: "Cinnabar Sunset", Length: sec(1521.34)},
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://thelovelymoon.bandcamp.com/album/echoes-of-memories",
						LinkType: seed.LinkType_DownloadForFree_Release_URL,
					},
					{
						URL:      "https://thelovelymoon.bandcamp.com/album/echoes-of-memories",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://thelovelymoon.bandcamp.com/album/echoes-of-memories",
						LinkType: seed.LinkType_StreamingMusic_Release_URL,
					},
					{
						URL:      "http://creativecommons.org/licenses/by-nc-sa/3.0/",
						LinkType: seed.LinkType_License_Release_URL,
					},
				},
			},
			img: "https://f4.bcbits.com/img/a2393960629_0.jpg",
		},
		{
			// This album is released by a label and isn't downloadable for free.
			url: "https://volaband.bandcamp.com/album/live-from-the-pool",
			rel: &seed.Release{
				Title:     "Live From The Pool",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Language:  "eng",
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Year: 2022, Month: 4, Day: 1, Country: "XW"}},
				Labels:    []seed.ReleaseLabel{{Name: "Mascot Label Group"}},
				Artists:   []seed.ArtistCredit{{Name: "VOLA"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "24 Light-Years (Live From The Pool)", Length: sec(322.773)},
						{Title: "Alien Shivers (Live From The Pool)", Length: sec(257.867)},
						{Title: "Head Mounted Sideways (Live From The Pool)", Length: sec(345.48)},
						{Title: "Straight Lines (Live From The Pool)", Length: sec(263.307)},
						{Title: "Ruby Pool (Live From The Pool)", Length: sec(272.947)},
						{Title: "Owls (Live From The Pool)", Length: sec(357.72)},
						{Title: "These Black Claws feat. SHAHMEN (Live From The Pool)", Length: sec(355.507)},
						{Title: "Gutter Moon (October Session) (Live From The Pool)", Length: sec(183.533)},
						{Title: "Ghosts (Live From The Pool)", Length: sec(245.493)},
						{Title: "Smartfriend (Live From The Pool)", Length: sec(256.6)},
						{Title: "Whaler (Live From The Pool)", Length: sec(344.68)},
						{Title: "Inside Your Fur (Live From The Pool)", Length: sec(338.24)},
						{Title: "Stray The Skies (Live From The Pool)", Length: sec(257.853)},
					},
				}},
				URLs: []seed.URL{
					{
						URL:      "https://volaband.bandcamp.com/album/live-from-the-pool",
						LinkType: seed.LinkType_PurchaseForDownload_Release_URL,
					},
					{
						URL:      "https://volaband.bandcamp.com/album/live-from-the-pool",
						LinkType: seed.LinkType_StreamingMusic_Release_URL,
					},
				},
			},
			img: "https://f4.bcbits.com/img/a1079469134_0.jpg",
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

func sec(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func TestCleanURL(t *testing.T) {
	var pr Provider
	for _, tc := range []struct {
		in   string
		want string
		ok   bool // if false, error should be returned
	}{
		{"https://louiezong.bandcamp.com/album/cartoon-funk", "https://louiezong.bandcamp.com/album/cartoon-funk", true},
		{"http://louiezong.bandcamp.com/album/cartoon-funk", "https://louiezong.bandcamp.com/album/cartoon-funk", true},
		{"https://pillarsinthesky.bandcamp.com/track/arcanum", "https://pillarsinthesky.bandcamp.com/track/arcanum", true},
		{"http://pillarsinthesky.bandcamp.com/track/arcanum", "https://pillarsinthesky.bandcamp.com/track/arcanum", true},
		{"http://user:pass@artist.bandcamp.com/album/name?foo=bar#hash", "https://artist.bandcamp.com/album/name", true},
		{"https://daily.bandcamp.com/best-jazz/the-best-jazz-on-bandcamp-october-2022", "", false},
		{"https://artist.example.org/album/name", "", false},
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
