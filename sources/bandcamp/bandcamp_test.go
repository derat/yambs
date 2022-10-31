// Copyright 2022 Daniel Erat.
// All rights reserved.

package bandcamp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/web"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

func TestParseAlbumPage(t *testing.T) {
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
				Artists:   []seed.ArtistCredit{{Name: "Louie Zong"}},
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
				EditNote: "https://louiezong.bandcamp.com/album/cartoon-funk" + editNote,
			},
			img: "https://f4.bcbits.com/img/a1689585732_0.jpg",
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
				Artists:   []seed.ArtistCredit{{Name: "The Lovely Moon"}},
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
				EditNote: "https://thelovelymoon.bandcamp.com/album/echoes-of-memories" + editNote,
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
				EditNote: "https://volaband.bandcamp.com/album/live-from-the-pool" + editNote,
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
			rel, img, err := parseAlbumPage(page, tc.url)
			if err != nil {
				t.Fatal("Failed parsing page:", err)
			}

			if diff := cmp.Diff(rel, tc.rel); diff != "" {
				t.Error("Bad release data:\n" + diff)
			}

			var imgURL string
			if img != nil {
				imgURL = img.URL()
			}
			if diff := cmp.Diff(imgURL, tc.img); diff != "" {
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