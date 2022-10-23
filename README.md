# yambs (Yet Another MusicBrainz Seeder)

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

To add more-complicated artist credits:

```
yambs \
  -type recording \
  -fields ... \
  -set artist0_mbid=1a054dd8-c5fa-40b6-9397-61c26b0185d4 \
  -set artist0_credited=virt \
  -set 'artist0_join_phrase= & ' \
  -set artist1_name=Rush \
  ...
```

To list names of available fields:

```
yambs -type recording -list-fields
```

## Further reading

*   ["Release Editor Seeding" documentation](https://wiki.musicbrainz.org/Development/Release_Editor_Seeding)
*   [Release seeding example](https://musicbrainz.org/static/tests/seed-love-bug.html)
*   ["Seeding Recordings" thread](https://community.metabrainz.org/t/seeding-recordings/188972)
*   [MusicBrainz userscripts](https://wiki.musicbrainz.org/Guides/Userscripts)
