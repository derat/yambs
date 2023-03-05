// Copyright 2022 Daniel Erat.
// All rights reserved.

package bandcamp

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online/internal"
	"github.com/derat/yambs/web"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

func TestRelease(t *testing.T) {
	ctx := context.Background()
	db := mbdb.NewDB(mbdb.DisallowQueries)
	var pr Provider

	// Add canned MBIDs for artists and label lookups.
	mkInfos := mbdb.MakeEntityInfosForTest
	for url, artists := range map[string][]mbdb.EntityInfo{
		"https://aartijadu.bandcamp.com/":       mkInfos("76cb3647-7a02-4f1d-9eef-8a6e99ce022d", "Aarti Jadu"),
		"https://anti-mass.bandcamp.com/":       mkInfos("7fa907c9-35a0-40dd-b926-8c962782ba1d", "ANTI-MASS"),
		"https://louiezong.bandcamp.com/":       mkInfos("0e2c603f-fd71-4ab6-af96-92c3e936586d", "louie zong"),
		"https://louiscole.bandcamp.com/":       mkInfos("525ef747-abf6-423c-98b4-cd49c0c07927", "Louis Cole"),
		"https://pillarsinthesky.bandcamp.com/": mkInfos("7ba8b326-34ba-472b-b710-b01dc1f14f94", "Pillars in the Sky"),
		"https://thelovelymoon.bandcamp.com/":   mkInfos("f34ae170-055d-46bc-9208-a750a646481b", "The Lovely Moon"),
		// "https://volaband.bandcamp.com/album/live-from-the-pool" omitted
	} {
		db.SetArtistsFromURLForTest(url, artists)
	}
	for url, labels := range map[string][]mbdb.EntityInfo{
		"https://brainfeeder.bandcamp.com/": mkInfos("20b3d6f9-9086-48d9-802f-5f808456a0ef", "Brainfeeder"),
		// "https://mascotlabelgroup.bandcamp.com/" omitted
		"https://syssistersounds.bandcamp.com/": mkInfos("2583b10d-0528-4281-8d2f-31d9a64a570c", "SYS Sister Sounds"),
	} {
		db.SetLabelsFromURLForTest(url, labels)
	}

	for _, tc := range []struct {
		url                 string
		extractTrackArtists bool
		rel                 *seed.Release
		img                 string
	}{
		{
			// This album was released on a subdomain that's linked to an artist, but the artist's
			// MBID shouldn't be seeded since the credited name is very different from the name in
			// the DB: https://github.com/derat/yambs/issues/26
			url: "https://aartijadu.bandcamp.com/track/just-eyes",
			rel: &seed.Release{
				Title:     "Just Eyes",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2019, 4, 6), Country: "XW"}},
				Artists:   []seed.ArtistCredit{{Name: "Aarti Jadu & Matthew Hayes"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Just Eyes", Length: sec(401.01)},
					},
				}},
				URLs: urlLinks("https://aartijadu.bandcamp.com/track/just-eyes",
					seed.LinkType_DownloadForFree_Release_URL,
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				),
			},
			img: "https://f4.bcbits.com/img/a3258735142_0.jpg",
		},
		{
			// This is a compilation album with artist name(s) at the beginning of each track.
			// Check that they get extracted when ExtractTrackArtists is true.
			url:                 "https://anti-mass.bandcamp.com/album/doxa",
			extractTrackArtists: true,
			rel: &seed.Release{
				Title:     "DOXA",
				Types:     nil,
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2021, 10, 7), Country: "XW"}},
				Artists: []seed.ArtistCredit{{
					MBID: "7fa907c9-35a0-40dd-b926-8c962782ba1d",
					Name: "ANTI-MASS",
				}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Galiba", Artists: artists("Authentically Plastic", " & ", "Nsasi"), Length: sec(230)},
						{Title: "Grind", Artists: artists("Nsasi"), Length: sec(156)},
						{Title: "Diesel Femme", Artists: artists("Turkana", " & ", "Authentically Plastic"), Length: sec(260)},
						{Title: "Influencer Convention", Artists: artists("Turkana"), Length: sec(240)},
						{Title: "Binia Yei", Artists: artists("Nsasi", " & ", "Turkana"), Length: sec(232)},
						{Title: "Sabula", Artists: artists("Authentically Plastic"), Length: sec(234)},
					},
				}},
				URLs: urlLinks("https://anti-mass.bandcamp.com/album/doxa",
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				),
			},
			img: "https://f4.bcbits.com/img/a2017435297_0.jpg",
		},
		{
			// This album has a hidden track, which should be included in the tracklist.
			// It also shouldn't have a stream-for-free link since the hidden track can't be streamed.
			url: "https://louiezong.bandcamp.com/album/cartoon-funk",
			rel: &seed.Release{
				Title:     "Cartoon Funk",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2022, 6, 17), Country: "XW"}},
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
				URLs: urlLinks("https://louiezong.bandcamp.com/album/cartoon-funk",
					seed.LinkType_DownloadForFree_Release_URL,
					seed.LinkType_PurchaseForDownload_Release_URL,
				),
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
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2022, 8, 2), Country: "XW"}},
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
				URLs: urlLinks("https://louiscole.bandcamp.com/album/let-it-happen",
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				),
			},
			img: "https://f4.bcbits.com/img/a3000320182_0.jpg",
		},
		{
			// This is a non-album track URL, which should be treated as a single.
			url: "https://pillarsinthesky.bandcamp.com/track/arcanum",
			rel: &seed.Release{
				Title:     "Arcanum",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2015, 5, 3), Country: "XW"}},
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
				URLs: urlLinks("https://pillarsinthesky.bandcamp.com/track/arcanum",
					seed.LinkType_DownloadForFree_Release_URL,
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				),
			},
			img: "https://f4.bcbits.com/img/a2320496643_0.jpg",
		},
		{
			// This is a non-album track with the artist's name prefixed to its title.
			// The artist name should be stripped even without setting ExtractTrackArtists.
			// The label MBID should also be derived from the hostname.
			url: "https://syssistersounds.bandcamp.com/track/apsara",
			rel: &seed.Release{
				Title:     "Apsara",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2022, 5, 18), Country: "XW"}},
				Labels:    []seed.ReleaseLabel{{MBID: "2583b10d-0528-4281-8d2f-31d9a64a570c"}},
				Artists:   []seed.ArtistCredit{{Name: "Maggie Tra"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Apsara", Length: sec(212.574)},
					},
				}},
				URLs: urlLinks("https://syssistersounds.bandcamp.com/track/apsara",
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				),
			},
			img: "https://f4.bcbits.com/img/a0574189382_0.jpg",
		},
		{
			// This album has a Creative Commons license.
			url: "https://thelovelymoon.bandcamp.com/album/echoes-of-memories",
			rel: &seed.Release{
				Title:     "Echoes of Memories",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2022, 10, 22), Country: "XW"}},
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
				URLs: append(urlLinks("https://thelovelymoon.bandcamp.com/album/echoes-of-memories",
					seed.LinkType_DownloadForFree_Release_URL,
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				), seed.URL{
					URL:      "http://creativecommons.org/licenses/by-nc-sa/3.0/",
					LinkType: seed.LinkType_License_Release_URL,
				}),
			},
			img: "https://f4.bcbits.com/img/a2393960629_0.jpg",
		},
		{
			// This album is released by a label and isn't downloadable for free.
			url: "https://volaband.bandcamp.com/album/live-from-the-pool",
			rel: &seed.Release{
				Title:     "Live From The Pool",
				Types:     []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Script:    "Latn",
				Status:    seed.ReleaseStatus_Official,
				Packaging: seed.ReleasePackaging_None,
				Events:    []seed.ReleaseEvent{{Date: seed.MakeDate(2022, 4, 1), Country: "XW"}},
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
				URLs: urlLinks("https://volaband.bandcamp.com/album/live-from-the-pool",
					seed.LinkType_PurchaseForDownload_Release_URL,
					seed.LinkType_FreeStreaming_Release_URL,
				),
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
			cfg := internal.Config{
				ExtractTrackArtists: tc.extractTrackArtists,
				DisallowNetwork:     true,
			}
			rel, img, err := pr.Release(ctx, page, tc.url, db, &cfg)
			if err != nil {
				t.Fatal("Failed parsing page:", err)
			}

			if diff := cmp.Diff(tc.rel, rel); diff != "" {
				t.Error("Bad release data:\n" + diff)
			}

			var imgURL string
			if img != nil {
				imgURL = img.URL("" /* serverURL */)
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

// artists constructs artist credits from a sequence of artist names and join phrases.
func artists(in ...string) []seed.ArtistCredit {
	credits := make([]seed.ArtistCredit, (len(in)+1)/2)
	for i := 0; i < len(in); i += 2 {
		credits[i/2].Name = in[i]
		if i+1 < len(in) {
			credits[i/2].JoinPhrase = in[i+1]
		}
	}
	return credits
}

// urlLinks constructs seed.URL objects for the specified URL and link types.
func urlLinks(url string, types ...seed.LinkType) []seed.URL {
	urls := make([]seed.URL, len(types))
	for i, t := range types {
		urls[i] = seed.URL{URL: url, LinkType: t}
	}
	return urls
}

func TestExtractTrackArtists(t *testing.T) {
	for _, tc := range []struct {
		orig, track string
		artists     []seed.ArtistCredit
	}{
		{"The Title", "The Title", nil},
		{"Artist Name - The Title", "The Title", artists("Artist Name")},
		{"Artist Name & Someone Else - The Title", "The Title", artists("Artist Name", " & ", "Someone Else")},
		{"Artist Name - The Title - More Junk", "The Title - More Junk", artists("Artist Name")},
		{" - The Title", " - The Title", nil},
		{"The Title - ", "The Title - ", nil},
	} {
		track, artists := extractTrackArtists(tc.orig)
		if track != tc.track || !reflect.DeepEqual(artists, tc.artists) {
			t.Errorf("extractTrackArtists(%q) = %q, %q; want %q, %q",
				tc.orig, track, artists, tc.track, tc.artists)
		}
	}
}

func TestParseArtists(t *testing.T) {
	for _, tc := range []struct {
		orig string
		want []seed.ArtistCredit
	}{
		{"Artist 1", artists("Artist 1")},
		{"Artist 1 & Artist 2", artists("Artist 1", " & ", "Artist 2")},
		{"Artist 1, Artist 2 & Artist 3", artists("Artist 1", ", ", "Artist 2", " & ", "Artist 3")},
		{"Artist 1 feat. Artist 2", artists("Artist 1", " feat. ", "Artist 2")},
		{"Artist 1 & Artist 2 feat. Artist 3", artists("Artist 1", " & ", "Artist 2", " feat. ", "Artist 3")},
		// Check that bad input is handled reasonably.
		{"Artist 1 & ", artists("Artist 1 & ")},
		{" & Artist 1", artists(" & Artist 1")},
		{" & ", artists(" & ")},
		{", & ", artists(", & ")},
		{",  & ", artists(",  & ")},
		{"", nil},
	} {
		if got := parseArtists(tc.orig); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseArtists(%q) = %+v; want %+v", tc.orig, got, tc.want)
		}
	}
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
