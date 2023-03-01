// Copyright 2023 Daniel Erat.
// All rights reserved.

package tidal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/derat/yambs/mbdb"
	"github.com/derat/yambs/seed"
	"github.com/derat/yambs/sources/online/internal"
	"github.com/google/go-cmp/cmp"
)

func TestGetRelease(t *testing.T) {
	ctx := context.Background()
	api := &fakeAPICaller{}
	now := time.Date(2015, 2, 10, 0, 0, 0, 0, time.UTC)

	db := mbdb.NewDB(mbdb.DisallowQueries)
	for url, mbid := range map[string]string{
		// Add both tidal.com and listen.tidal.com URLs since both appear in the database.
		"https://tidal.com/artist/608":            "4bd95eea-b9f6-4d70-a36c-cfea77431553",
		"https://listen.tidal.com/artist/9091":    "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd",
		"https://tidal.com/artist/24905":          "309c62ba-7a22-4277-9f67-4a162526d18a",
		"https://listen.tidal.com/artist/5483069": "65b1de19-50cb-49fe-b802-d1d8616f9ebe",
	} {
		db.SetArtistMBIDFromURLForTest(url, mbid)
	}

	for _, tc := range []struct {
		url     string
		country string // defaultCountry if empty
		rel     *seed.Release
		img     string
	}{
		{
			// This request should fail: the API reports that the album has 16 tracks, but the
			// /tracks endpoint only returns 9 for US (presumably due to country restrictions).
			url: "https://tidal.com/album/1588184",
			rel: nil,
			img: "",
		},
		{
			// When querying all countries, the two countries with the full tracklist should
			// be mentioned in the annotation.
			url:     "https://tidal.com/album/1588184",
			country: "XW",
			rel: &seed.Release{
				Title: "Step Up 2 The Streets Original Motion Picture Soundtrack",
				Types: []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Annotation: "© 2008 Atlantic Recording Corporation for the United States and " +
					"WEA International Inc. for the world outside of the United States.\n\n" +
					"Regions with all tracks on Tidal (as of 2015-02-10 UTC):\n" +
					"    * Norway (NO)\n" +
					"    * Sweden (SE)",
				Barcode:   "075679994264",
				Script:    "Latn",
				Status:    "Official",
				Packaging: "None",
				Labels:    []seed.ReleaseLabel{{Name: "Atlantic"}},
				Artists:   []seed.ArtistCredit{{Name: "Step Up 2 The Streets"}},
				URLs: []seed.URL{{
					URL:      "https://tidal.com/album/1588184",
					LinkType: seed.LinkType_Streaming_Release_URL,
				}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						track("Low (feat. T-Pain) [Step Up 2 the Streets O.S.T. Version]", sec(230), "Flo Rida", " feat. ", "T-Pain"),
						track("Shake Your Pom Pom", sec(240), "Missy Elliott"),
						track("Killa", sec(231), "Cherish featuring Yung Joc"),
						track("Hypnotized (feat. Akon) [Step up 2 the Streets Original Soundtrack Version]", sec(188),
							"Plies", " feat. ", "Akon"),
						track("Is It You (Step Up 2 the Streets O.S.T. Version)", sec(238), "Cassie"),
						track("Can't Help but Wait", sec(205), "Trey Songz"),
						track("Church (feat. Teddy Verseti) [Step Up 2 the Streets Original Soundtrack Version]", sec(241),
							"T-Pain", " feat. ", "Teddy Verseti"),
						track("Ching-A-Ling", sec(219), "Missy Elliott"),
						track("Push (Step Up 2 the Streets O.S.T. Version)", sec(208), "Enrique Iglesias"),
						track("369 (feat. B.o.B.) [Step up 2 the Streets Original Soundtrack Version]", sec(211), "Cupid", " & ", "B.o.B"),
						track("Impossible (Step Up 2 the Streets O.S.T. Version)", sec(218), "Bayje"),
						track("Lives in da Club (feat. Jay Lyriq) [Step Up 2 the Streets O.S.T. Version]", sec(209),
							"Sophia Fresh", " feat. ", "Jay Lyriq"),
						track("Girl You Know (Feat. Trey Songz) [Step Up 2 The Streets O.S.T. Version]", sec(254), "Scarface"),
						track("Say Cheese (Step Up 2 the Streets O.S.T. Version)", sec(245), "K.C."),
						track("Let It Go (Step Up 2 the Streets O.S.T. Version)", sec(202), "Brit & Alex"),
						track("Ain't No Stressin (Step Up 2 the Streets O.S.T. Version)", sec(260), "Montana Tucker, Sikora, Denial"),
					},
				}},
			},
			img: "https://resources.tidal.com/images/ec479657/8559/45a2/9d67/42a5f8d4c048/origin.jpg",
		},
		{
			url: "https://tidal.com/album/24700142",
			rel: &seed.Release{
				Title:      "Sap",
				Types:      []seed.ReleaseGroupType{seed.ReleaseGroupType_EP},
				Annotation: "(P) 1992 Sony Music Entertainment",
				Barcode:    "884977869965",
				Script:     "Latn",
				Status:     seed.ReleaseStatus_Official,
				Packaging:  seed.ReleasePackaging_None,
				Labels:     []seed.ReleaseLabel{{Name: "Columbia"}, {Name: "Sbme Special Mkts."}},
				Artists:    []seed.ArtistCredit{{Name: "Alice In Chains", MBID: "4bd95eea-b9f6-4d70-a36c-cfea77431553"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Brother", Length: sec(267)},
						{Title: "Got Me Wrong", Length: sec(250)},
						{Title: "Right Turn", Length: sec(194)},
						{Title: "Am I Inside", Length: sec(308)},
						{Title: "Love Song", Length: sec(225)},
					},
				}},
				URLs: []seed.URL{{
					URL:      "https://tidal.com/album/24700142",
					LinkType: seed.LinkType_Streaming_Release_URL,
				}},
			},
			img: "https://resources.tidal.com/images/d546be20/d268/46ab/874c/29364764407f/origin.jpg",
		},
		{
			url: "https://tidal.com/album/58823194",
			rel: &seed.Release{
				Title:      "Junk",
				Types:      []seed.ReleaseGroupType{seed.ReleaseGroupType_Album},
				Annotation: "Copyright 2016 M83 Recording Inc. under exclusive license to Mute for North America",
				Barcode:    "724596964057",
				Script:     "Latn",
				Status:     seed.ReleaseStatus_Official,
				Packaging:  seed.ReleasePackaging_None,
				Events:     []seed.ReleaseEvent{{Date: seed.MakeDate(2016, 4, 8)}},
				Labels:     []seed.ReleaseLabel{{Name: "Mute"}},
				Artists:    []seed.ArtistCredit{{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{
						{Title: "Do It, Try It", Length: sec(217)},
						{Title: "Go! (feat. Mai Lan)", Length: sec(236), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "Mai Lan", MBID: "65b1de19-50cb-49fe-b802-d1d8616f9ebe"},
						}},
						{Title: "Walkway Blues (feat. J Laser)", Length: sec(289), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "J Laser"},
						}},
						{Title: "Bibi The Dog (feat. Mai Lan)", Length: sec(234), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "Mai Lan", MBID: "65b1de19-50cb-49fe-b802-d1d8616f9ebe"},
						}},
						{Title: "Moon Crystal", Length: sec(147)},
						{Title: "For The Kids (feat. Susanne Sundfør)", Length: sec(281), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "Susanne Sundfør"},
						}},
						{Title: "Solitude", Length: sec(364)},
						{Title: "The Wizard", Length: sec(145)},
						{Title: "Laser Gun (feat. Mai Lan)", Length: sec(257), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "Mai Lan", MBID: "65b1de19-50cb-49fe-b802-d1d8616f9ebe"},
						}},
						{Title: "Road Blaster", Length: sec(262)},
						{Title: "Tension", Length: sec(126)},
						{Title: "Atlantique Sud (feat. Mai Lan)", Length: sec(204), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "Mai Lan", MBID: "65b1de19-50cb-49fe-b802-d1d8616f9ebe"},
						}},
						{Title: "Time Wind (feat. Beck)", Length: sec(249), Artists: []seed.ArtistCredit{
							{Name: "M83", MBID: "6d7b7cd4-254b-4c25-83f6-dd20f98ceacd", JoinPhrase: " feat. "},
							{Name: "Beck", MBID: "309c62ba-7a22-4277-9f67-4a162526d18a"},
						}},
						{Title: "Ludivine", Length: sec(95)},
						{Title: "Sunday Night 1987", Length: sec(240)},
					},
				}},
				URLs: []seed.URL{{
					URL:      "https://tidal.com/album/58823194",
					LinkType: seed.LinkType_Streaming_Release_URL,
				}},
			},
			img: "https://resources.tidal.com/images/00355f8a/1727/49e4/a2c9/738404c005e9/origin.jpg",
		},
		{
			url: "https://tidal.com/album/93071188",
			rel: &seed.Release{
				Title:      "Never Fade",
				Types:      []seed.ReleaseGroupType{seed.ReleaseGroupType_Single},
				Annotation: "© 2018 AIC Entertainment, LLC under exclusive license to BMG Rights Management (US) LLC",
				Barcode:    "4050538433197",
				Script:     "Latn",
				Status:     seed.ReleaseStatus_Official,
				Packaging:  seed.ReleasePackaging_None,
				Events:     []seed.ReleaseEvent{{Date: seed.MakeDate(2018, 8, 10)}},
				Labels:     []seed.ReleaseLabel{{Name: "BMG"}},
				Artists:    []seed.ArtistCredit{{Name: "Alice In Chains", MBID: "4bd95eea-b9f6-4d70-a36c-cfea77431553"}},
				Mediums: []seed.Medium{{
					Format: seed.MediumFormat_DigitalMedia,
					Tracks: []seed.Track{{Title: "Never Fade", Length: sec(280)}},
				}},
				URLs: []seed.URL{{
					URL:      "https://tidal.com/album/93071188",
					LinkType: seed.LinkType_Streaming_Release_URL,
				}},
			},
			img: "https://resources.tidal.com/images/f6a26633/8c97/4bce/9d29/a3f4ed9637e3/origin.jpg",
		},
		{
			// This request should fail: the album is completely unavailable in the US.
			url: "https://tidal.com/album/251633624",
			rel: nil,
			img: "",
		},
		{
			// When querying all countries, NO (where the album is available) should be used
			// for the album and credits queries: https://github.com/derat/yambs/issues/25
			url:     "https://tidal.com/album/251633624",
			country: "XW",
			rel: &seed.Release{MBID: "",
				Title: "Like Drawing Blood (Deluxe Version)",
				Types: []seed.ReleaseGroupType{"Album"},
				Annotation: "Copyright Lucky Number Music Limited\n\n" +
					"Regions with all tracks on Tidal (as of 2015-02-10 UTC):\n" +
					"    * Norway (NO)",
				Barcode:   "5025425173233",
				Script:    "Latn",
				Status:    "Official",
				Packaging: "None",
				Artists:   []seed.ArtistCredit{{Name: "Gotye"}},
				Mediums: []seed.Medium{{
					Format: "Digital Media",
					Tracks: []seed.Track{
						track("Like Drawing Blood", sec(22)),
						track("The Only Way", sec(284)),
						track("Hearts A Mess", sec(365)),
						track("Coming Back", sec(360)),
						track("Thanks For Your Time", sec(260)),
						track("Learnalilgivinanlovin", sec(173)),
						track("Puzzle With A Piece Missing", sec(341)),
						track("Seven Hours With A Backseat Driver", sec(283)),
						track("The Only Thing I Know", sec(423)),
						track("Night Drive", sec(310)),
						track("Worn Out Blues", sec(38)),
						track("Coming Back", sec(201), "Gotye", " & ", "Inga Liljestrom"),
						track("Hearts A Mess", sec(499)),
						track("Puzzle With A Piece Missing", sec(261)),
						track("Learnalilgivinanlovin", sec(275)),
						track("Thanks For Your Time", sec(252)),
					},
				}},
				URLs: []seed.URL{{
					URL:      "https://tidal.com/album/251633624",
					LinkType: seed.LinkType_Streaming_Release_URL,
				}},
			},
			img: "https://resources.tidal.com/images/32aaa3ef/9ae6/4f3b/ad8e/a68562087836/origin.jpg",
		},
	} {
		t.Run(tc.url, func(t *testing.T) {
			cfg := &internal.Config{
				DisallowNetwork: true,
				CountryCode:     tc.country,
			}
			rel, img, err := getRelease(ctx, tc.url, api, db, cfg, now)
			if tc.rel == nil {
				if err == nil {
					t.Fatal("Expected error but unexpectedly succeeded")
				}
				return
			}

			if err != nil {
				t.Fatal("Failed getting release:", err)
			}
			if diff := cmp.Diff(tc.rel, rel); diff != "" {
				t.Error("Bad release data:\n" + diff)
				// The next line can be uncommented when adding a new test to dump the raw struct.
				//fmt.Printf("%#v\n", rel)
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

func sec(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

func track(title string, length time.Duration, creds ...string) seed.Track {
	tr := seed.Track{Title: title, Length: length}
	for i := 0; i < len(creds); i += 2 {
		ac := seed.ArtistCredit{Name: creds[i]}
		if i+1 < len(creds) {
			ac.JoinPhrase = creds[i+1]
		}
		tr.Artists = append(tr.Artists, ac)
	}
	return tr
}

type fakeAPICaller struct{}

func (*fakeAPICaller) call(ctx context.Context, path string) ([]byte, error) {
	read := func(p string) ([]byte, error) {
		f, err := os.Open(p)
		if os.IsNotExist(err) {
			return nil, notFoundErr
		} else if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}

	if ms := apiAlbumRegexp.FindStringSubmatch(path); ms != nil {
		return read(filepath.Join("testdata", "album_"+ms[1]+"_"+ms[2]+".json"))
	} else if ms := apiCreditsRegexp.FindStringSubmatch(path); ms != nil {
		return read(filepath.Join("testdata", "credits_"+ms[1]+"_"+ms[2]+".json"))
	} else if ms := apiTracksRegexp.FindStringSubmatch(path); ms != nil {
		return read(filepath.Join("testdata", "tracks_"+ms[1]+"_"+ms[2]+".json"))
	}
	return nil, fmt.Errorf("unhandled path %q", path)
}

// These match API paths requested by getRelease().
var apiAlbumRegexp = regexp.MustCompile(`^/v1/albums/(\d+)\?countryCode=([A-Z]{2})$`)
var apiCreditsRegexp = regexp.MustCompile(`^/v1/albums/(\d+)/credits\?countryCode=([A-Z]{2})$`)
var apiTracksRegexp = regexp.MustCompile(`^/v1/albums/(\d+)/tracks\?countryCode=([A-Z]{2})$`)

func TestMakeArtistCredits(t *testing.T) {
	ctx := context.Background()
	db := mbdb.NewDB(mbdb.DisallowQueries)
	for _, tc := range []struct {
		artists []artistData
		want    []seed.ArtistCredit
	}{
		{
			artists: []artistData{{ID: 1, Name: "Artist", Type: "MAIN"}},
			want:    []seed.ArtistCredit{{Name: "Artist"}},
		},
		{
			artists: []artistData{
				{ID: 1, Name: "A", Type: "MAIN"},
				{ID: 2, Name: "B", Type: "MAIN"},
				{ID: 3, Name: "C", Type: "MAIN"},
				{ID: 4, Name: "D", Type: "FEATURED"},
			},
			want: []seed.ArtistCredit{
				{Name: "A", JoinPhrase: ", "},
				{Name: "B", JoinPhrase: " & "},
				{Name: "C", JoinPhrase: " feat. "},
				{Name: "D"},
			},
		},
	} {
		got := makeArtistCredits(ctx, tc.artists, db)
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("Bad artist credits from %v:\n%v", tc.artists, diff)
		}
		// TODO: Also check that the negative cache is used properly.
	}
}

func TestCleanURL(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool // if false, error should be returned
	}{
		{"https://tidal.com/album/12345", "https://tidal.com/album/12345", true},
		{"http://tidal.com/album/12345", "https://tidal.com/album/12345", true},
		{"https://tidal.com/album/12345?utm_source=google#foo", "https://tidal.com/album/12345", true},
		{"https://listen.tidal.com/album/12345", "https://tidal.com/album/12345", true},
		{"https://tidal.com/browse/album/12345", "https://tidal.com/album/12345", true},
		{"https://help.tidal.com/album/12345", "", false},
		{"https://example.com/album/12345", "", false},
		{"https://tidal.com/album/bogus", "", false},
	} {
		if got, err := cleanURL(tc.in); !tc.ok && err == nil {
			t.Errorf("cleanURL(%q) = %q; wanted error", tc.in, got)
		} else if tc.ok && err != nil {
			t.Errorf("cleanURL(%q) failed: %v", tc.in, err)
		} else if tc.ok && got != tc.want {
			t.Errorf("cleanURL(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
