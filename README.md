# yambs (Yet Another MusicBrainz Seeder)

[![Build Status](https://storage.googleapis.com/derat-build-badges/666a3806-8543-471a-bd6c-d4b154f96082.svg)](https://storage.googleapis.com/derat-build-badges/666a3806-8543-471a-bd6c-d4b154f96082.html)

`yambs` is a command-line program for seeding edits to the [MusicBrainz] music
database.

[MusicBrainz]: https://musicbrainz.org/

## Usage

To compile and install the `yambs` executable, install [Go] and run the
following command:

```sh
go install ./cmd/yambs
```

[Go]: https://go.dev/

```
Usage yambs: [flag]... <FILE/URL>
Seeds MusicBrainz edits.

  -action value
    	Action to perform with seed URLs (open, print, serve, write) (default open)
  -addr string
    	Address to listen on for HTTP requests (default "localhost:8999")
  -fields string
    	Comma-separated fields for CSV/TSV columns (e.g. "artist,title,length")
  -format value
    	Format for text input (csv, keyval, tsv) (default tsv)
  -list-fields
    	Print available fields for -type and exit
  -set value
    	Set a field for all entities (e.g. "artist=The Beatles")
  -type value
    	Type of entity to edit (recording, release) (default recording)
```

To add multiple non-album recordings for a single artist, you can run a command
like the following:

```sh
yambs \
  -type recording \
  -fields title,length,edit_note \
  -set artist=b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d \
  <recordings.tsv
```

with a `recordings.tsv` file like the following:

```tsv
Song #1	4:35	info from https://example.org/song1.html
Song #2	53.234	info from https://example.org/song2.html
```

To edit existing recordings, specify their [MBID]s via the `mbid` field:

```sh
yambs \
  -type recording \
  -fields mbid,title \
  <recordings.tsv
```

```tsv
c55e74ff-bd7d-40ff-a591-c6993c59bda8	Sgt. Pepper’s Lonely Hearts Club Band
...
```

To add more-complicated artist credits:

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

The `keyval` format can be used to specify a single entity across multiple
lines:

```
yambs \
  -type recording \
  -format keyval \
  <recording.txt
```

with a `recording.txt` file like the following:

```txt
artist_credited=Some Artist
title=Test Title
length=3:56
edit_note=from http://www.example.org
```

To list all available fields:

```
yambs -type recording -list-fields
```

[MBID]: https://musicbrainz.org/doc/MusicBrainz_Identifier

## Further reading

*   ["Release Editor Seeding" documentation](https://wiki.musicbrainz.org/Development/Release_Editor_Seeding)
*   [Release seeding example](https://musicbrainz.org/static/tests/seed-love-bug.html)
*   ["Seeding Recordings" thread](https://community.metabrainz.org/t/seeding-recordings/188972)
*   [MusicBrainz userscripts](https://wiki.musicbrainz.org/Guides/Userscripts)
