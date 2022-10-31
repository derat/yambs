# yambs (Yet Another MusicBrainz Seeder)

[![Build Status](https://storage.googleapis.com/derat-build-badges/666a3806-8543-471a-bd6c-d4b154f96082.svg)](https://storage.googleapis.com/derat-build-badges/666a3806-8543-471a-bd6c-d4b154f96082.html)

`yambs` is a command-line program for seeding edits to the [MusicBrainz] music
database.

It's useful if you're trying to add a bunch of standalone recordings: given a
[CSV] or [TSV] file describing recordings, `yambs` can open the [Add Standalone
Recording] page for each with various fields pre-filled.

`yambs` can also read `key=value` lines from text files to seed the [Add
Release] page, and it can use [Bandcamp] album pages to seed releases too.

[MusicBrainz]: https://musicbrainz.org/
[CSV]: https://en.wikipedia.org/wiki/Comma-separated_values
[TSV]: https://en.wikipedia.org/wiki/Tab-separated_values
[Add Standalone Recording]: https://musicbrainz.org/recording/create
[Add Release]: http://musicbrainz.org/release/add
[Bandcamp]: https://bandcamp.com/

## Usage

To compile and install the `yambs` executable, install [Go] and run the
following command:

```sh
go install ./cmd/yambs
```

[Go]: https://go.dev/

---

```
Usage yambs: [flag]... <FILE/URL>
Seeds MusicBrainz edits.

  -action value
    	Action to perform with seed URLs (open, print, serve, write) (default open)
  -addr string
    	Address to listen on for HTTP requests (default "localhost:8999")
  -fields string
    	Comma-separated fields for CSV/TSV columns (e.g. "artist,name,length")
  -format value
    	Format for text input (csv, keyval, tsv) (default tsv)
  -list-fields
    	Print available fields for -type and exit
  -set value
    	Set a field for all entities (e.g. "artist=The Beatles")
  -type value
    	Type of entity to edit (recording, release) (default recording)
  -version
    	Print the version and exit
```

---

To add multiple non-album recordings for a single artist, you can run a command
like the following:

```sh
yambs \
  -type recording \
  -fields name,length,edit_note \
  -set artist=7e84f845-ac16-41fe-9ff8-df12eb32af55 \
  <recordings.tsv
```

with a `recordings.tsv` file like the following (with tab characters between the
fields):

```tsv
Song #1	4:35	info from https://example.org/song1.html
Song #2	53234.35	info from https://example.org/song2.html
```

The recordings' names, lengths, and edit notes will be read from the TSV file,
and the `-set artist=...` flag sets all recordings' `artist` field to the
[specified artist](https://musicbrainz.org/artist/7e84f845-ac16-41fe-9ff8-df12eb32af55).

---

To edit existing recordings, specify their [MBID]s via the `mbid` field:

```sh
yambs \
  -type recording \
  -fields mbid,name \
  <recordings.tsv
```

```tsv
c55e74ff-bd7d-40ff-a591-c6993c59bda8	Sgt. Pepper’s Lonely Hearts Club Band
...
```

---

More-complicated artist credits can also be assigned:

```
yambs \
  -type recording \
  -fields ... \
  -set artist0_mbid=1a054dd8-c5fa-40b6-9397-61c26b0185d4 \
  -set artist0_credited=virt \
  -set 'artist0_join= & ' \
  -set artist1_name=Rush \
  ...
```

(Note that repeated fields are 0-indexed.)

---

The `keyval` format can be used to seed a single entity across multiple lines:

```
yambs \
  -type release \
  -format keyval \
  <release.txt
```

with a `release.txt` file like the following:

```txt
title=Some Album
artist0_name=Some Artist
types=Album,Soundtrack
status=Official
packaging=Jewel Case
language=eng
script=Latn
event0_date=2021-05-15
event0_country=XW
medium0_format=CD
medium0_track0_title=First Track
medium0_track0_length=3:45.04
medium0_track1_title=Second Track
medium1_format=CD
medium1_track0_title=First Track on Second Disc
url0_url=https://www.example.org/
url0_type=75
edit_note=https://www.example.org
```

---

Pass the `-list-fields` flag to list all available fields for a given entity
type:

```
yambs -type recording -list-fields
yambs -type release   -list-fields
```

Acceptable values for various fields are listed in
[seed/enums.go](./seed/enums.go), which is automatically generated from
[t/sql/initial.sql](https://github.com/metabrainz/musicbrainz-server/blob/master/t/sql/initial.sql)
in the [musicbrainz-server](https://github.com/metabrainz/musicbrainz-server/)
repository.

[MBID]: https://musicbrainz.org/doc/MusicBrainz_Identifier

## Why?

There are a bunch of [MusicBrainz userscripts] that run in the browser with the
help of an extension like [Tampermonkey] to seed edits. They're well-tested, so
why not just use them instead of writing a new thing?

Well, at first I was adding a bunch of standalone recordings that I'd downloaded
from random musicians' homepages. I couldn't find any userscripts to help with
that, since the main focus seems to be seeding releases from major websites. I
ended up hacking together a shell script to generate URLs that would seed my
edits, but I figured it'd be nice to have something more robust and convenient
to use next time.

I had also been using the [bandcamp_importer.user.js] userscript to import
releases from Bandcamp, but I'm nervous about using extensions like Tampermonkey
that require permission to modify data on all sites. I'm not so worried about
malice on the part of extension or userscript developers, but I have no idea
about their security practices and I'm fearful of attackers compromising their
computers and uploading malicious versions of their code.

I created a separate browser profile that I could use to run Tampermonkey
without exposing any of my (non-MusicBrainz) credentials, but using it was a
pain, so I decided to add Bandcamp support to this codebase as well since
[that's where I get most of my music](https://www.erat.org/buying_music.html).

[MusicBrainz userscripts]: https://wiki.musicbrainz.org/Guides/Userscripts
[Tampermonkey]: https://www.tampermonkey.net/
[bandcamp_importer.user.js]: https://github.com/murdos/musicbrainz-userscripts/blob/master/bandcamp_importer.user.js

## Further reading

*   ["Release Editor Seeding" documentation](https://wiki.musicbrainz.org/Development/Release_Editor_Seeding)
*   [Release seeding example](https://musicbrainz.org/static/tests/seed-love-bug.html)
*   ["Seeding Recordings" thread](https://community.metabrainz.org/t/seeding-recordings/188972)
