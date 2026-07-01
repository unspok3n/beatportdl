# BeatportDL

Beatport & Beatsource downloader (FLAC, AAC)

_Requires an active [Beatport](https://stream.beatport.com/) or [Beatsource](https://stream.beatsource.com/) streaming plan._

![Screenshot](/screenshots/main.png?raw=true "Screenshot")

## Setup

1. [Download](https://github.com/unspok3n/beatportdl/releases/) or [build](#building) BeatportDL.

   _Compiled binaries for Windows, macOS (amd64, arm64) and Linux (amd64, arm64) are available on the [Releases](https://github.com/unspok3n/beatportdl/releases) page._ \
    _Don't forget to set the execute permission on unix systems, e.g., chmod +x beatportdl-darwin-arm64_

2. Run beatportdl (e.g. `./beatportdl-darwin-arm64`), then specify the:

   - Beatport username
   - Beatport password
   - Downloads directory
   - Audio quality

3. OPTIONAL: Customize a config file. Create a new config file by running:

```shell
./beatportdl
```

This will create a new `beatportdl-config.yml` file. You can put the following options and values into the config file:

---

| Option                        | Default Value                             | Type       | Description                                                                                                                                                                               |
| ----------------------------- | ----------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `username`                    |                                           | String     | Beatport username                                                                                                                                                                         |
| `password`                    |                                           | String     | Beatport password                                                                                                                                                                         |
| `quality`                     | lossless                                  | String     | Download quality _(medium-hls, medium, high, lossless)_                                                                                                                                   |
| `show_progress`               | true                                      | Boolean    | Enable progress bars                                                                                                                                                                      |
| `write_error_log`             | false                                     | Boolean    | Write errors to `error.log`                                                                                                                                                               |
| `max_download_workers`        | 15                                        | Integer    | Concurrent download jobs limit                                                                                                                                                            |
| `max_global_workers`          | 15                                        | Integer    | Concurrent global jobs limit                                                                                                                                                              |
| `downloads_directory`         |                                           | String     | Location for the downloads directory                                                                                                                                                      |
| `sort_by_context`             | false                                     | Boolean    | Create a directory for each release, playlist, chart, label, or artist                                                                                                                    |
| `sort_by_label`               | false                                     | Boolean    | Use label names as parent directories for releases (requires `sort_by_context`)                                                                                                           |
| `force_release_directories`   | false                                     | Boolean    | Create release directories inside chart and playlist folders (requires `sort_by_context`)                                                                                                 |
| `track_exists`                | update                                    | String     | Behavior when track file already exists                                                                                                                                                   |
| `track_number_padding`        | 2                                         | Integer    | Track number padding for filenames and tag mappings (when using `track_number_with_padding` or `release_track_count_with_padding`)<br/> Set to 0 for dynamic padding based on track count |
| `cover_size`                  | 1400x1400                                 | String     | Cover art size for `keep_cover` and track metadata (if `fix_tags` is enabled) _[max: 1400x1400]_                                                                                          |
| `keep_cover`                  | false                                     | Boolean    | Download cover art file (cover.jpg) to the context directory (requires `sort_by_context`)                                                                                                 |
| `fix_tags`                    | true                                      | Boolean    | Enable tag writing capabilities                                                                                                                                                           |
| `tag_mappings`                | _Listed below_                            | String Map | Custom tag mappings                                                                                                                                                                       |
| `track_file_template`         | {number}. {artists} - {name} ({mix_name}) | String     | Track filename template                                                                                                                                                                   |
| `release_directory_template`  | [{catalog_number}] {artists} - {name}     | String     | Release directory template                                                                                                                                                                |
| `playlist_directory_template` | {name} [{created_date}]                   | String     | Playlist directory template                                                                                                                                                               |
| `chart_directory_template`    | {name} [{published_date}]                 | String     | Chart directory template                                                                                                                                                                  |
| `label_directory_template`    | {name} [{updated_date}]                   | String     | Label directory template                                                                                                                                                                  |
| `artist_directory_template`   | {name}                                    | String     | Artist directory template                                                                                                                                                                 |
| `whitespace_character`        |                                           | String     | Whitespace character for track filenames and release directories                                                                                                                          |
| `artists_limit`               | 3                                         | Integer    | Maximum number of artists allowed before replacing with `artists_short_form` (affects directories, filenames, and search results)                                                         |
| `artists_short_form`          | VA                                        | String     | Custom string to represent "Various Artists"                                                                                                                                              |
| `key_system`                  | standard-short                            | String     | Music key system used in filenames and tags                                                                                                                                               |
| `proxy`                       |                                           | String     | Proxy URL                                                                                                                                                                                 |

If the Beatport credentials are correct, you should also see the file `beatportdl-credentials.json` appear in the BeatportDL directory.
_If you accidentally entered an incorrect password and got an error, you can always manually edit the config file_

Download quality options, per Beatport/Beatsource subscription type:

| Option       | Description                                                                                                  | Requires at least              | Notes                                                                   |
| ------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------ | ----------------------------------------------------------------------- |
| `medium-hls` | 128 kbps AAC through `/stream` endpoint (IMPORTANT: requires [ffmpeg](https://www.ffmpeg.org/download.html)) | Essential / Beatsource         | Same as `medium` on Advanced but uses a slightly slower download method |
| `medium`     | 128 kbps AAC                                                                                                 | Advanced / Beatsource Pro+     |                                                                         |
| `high`       | 256 kbps AAC                                                                                                 | Professional / Beatsource Pro+ |                                                                         |
| `lossless`   | 44.1 kHz FLAC                                                                                                | Professional / Beatsource Pro+ |                                                                         |

Available `track_exists` options:

- `error` Log error and skip
- `skip` Skip silently
- `overwrite` Re-download
- `update` Update tags

Available template keywords for filenames and directories (`*_template`):

- Track: `id`,`name`,`mix_name`,`slug`,`artists`,`remixers`,`number`,`length`,`key`,`bpm`,`genre`,`subgenre`,`genre_with_subgenre`,`subgenre_or_genre`,`isrc`,`label`
- Release: `id`,`name`,`slug`,`artists`,`remixers`,`date`,`year`,`track_count`,`bpm_range`,`catalog_number`,`upc`,`label`
- Playlist: `id`,`name`,`first_genre`,`track_count`,`bpm_range`,`length`,`created_date`,`updated_date`
- Chart: `id`,`name`,`slug`,`first_genre`,`track_count`,`creator`,`created_date`,`published_date`,`updated_date`

**Note:** The `first_genre` parameter for playlists and charts now uses the genre of the first track in the playlist/chart, rather than relying on metadata. This ensures more accurate genre-based folder organization.

- Artist: `id`, `name`, `slug`
- Label: `id`, `name`, `slug`, `created_date`, `updated_date`

Default `tag_mappings` config:

```yaml
tag_mappings:
  flac:
    track_name: "TITLE"
    track_artists: "ARTIST"
    track_number: "TRACKNUMBER"
    track_subgenre_or_genre: "GENRE"
    track_key: "KEY"
    track_bpm: "BPM"
    track_isrc: "ISRC"

    release_name: "ALBUM"
    release_artists: "ALBUMARTIST"
    release_date: "DATE"
    release_track_count: "TOTALTRACKS"
    release_catalog_number: "CATALOGNUMBER"
    release_label: "LABEL"
  m4a:
    track_name: "TITLE"
    track_artists: "ARTIST"
    track_number: "TRACKNUMBER"
    track_genre: "GENRE"
    track_key: "KEY"
    track_bpm: "BPM"
    track_isrc: "ISRC"

    release_name: "ALBUM"
    release_artists: "ALBUMARTIST"
    release_date: "DATE"
    release_track_count: "TOTALTRACKS"
    release_catalog_number: "CATALOGNUMBER"
    release_label: "LABEL"
```

As you can see, each key here represents a predefined value from either a release or a track that you can use to customize what is written to which tags. When you add an entry in the mappings for any format (for e.g., `flac`), only the tags that you specify will be written.

All tags by default are converted to uppercase, but since some M4A players might not recognize it, you can write the tag in lowercase and add the `_raw` suffix to bypass the conversion. _(This applies to M4A tags only)_

For e.g., Traktor doesn't recognize the track key tag in uppercase, so you have to add:

```yaml
tag_mappings:
  m4a:
    track_key: "initialkey_raw"
```

Available `tag_mappings` keys: `track_id`,`track_url`,`track_name`,`track_artists`,`track_artists_limited`,`track_remixers`,`track_remixers_limited`,`track_number`,`track_number_with_padding`,`track_number_with_total`,`track_genre`,`track_subgenre`,`track_genre_with_subgenre`,`track_subgenre_or_genre`,`track_key`,`track_bpm`,`track_isrc`,`release_id`,`release_url`,`release_name`,`release_artists`,`release_artists_limited`,`release_remixers`,`release_remixers_limited`,`release_date`,`release_year`,`release_track_count`,`release_track_count_with_padding`,`release_catalog_number`,`release_upc`,`release_label`,`release_label_url`

Available `key_system` options:

| System           | Example           |
| ---------------- | ----------------- |
| `standard`       | Eb Minor, F Major |
| `standard-short` | Ebm, F            |
| `openkey`        | 7m, 12d           |
| `camelot`        | 2A, 7B            |

Proxy URL format example: `http://username:password@127.0.0.1:8080`

## Usage

Run BeatportDL and enter Beatport or Beatsource URL or search query:

```shell
./beatportdl
Enter url or search query:
```

By default, search returns the results from beatport, if you want to search on beatsource instead, include `@beatsource` tag in the query

...or specify the URL using positional arguments:

```shell
./beatportdl https://www.beatport.com/track/strobe/1696999 https://www.beatport.com/track/move-for-me/591753
```

...or provide a text file with urls (separated by a newline)

```shell
./beatportdl file.txt file2.txt
```

URL types that are currently supported: **Tracks, Releases, Playlists, Charts, Labels, Artists**

## Building

Required dependencies:

- [TagLib](https://github.com/taglib/taglib) >= 2.0
- [zlib](https://github.com/madler/zlib) >= 1.2.3
- [Zig C/C++ Toolchain](https://github.com/ziglang/zig) >= 0.14.0

BeatportDL uses [TagLib](https://taglib.org/) C bindings to handle audio metadata and therefore requires [CGO](https://go.dev/wiki/cgo)

Makefile is adapted for cross-compilation and uses [Zig toolchain](https://github.com/ziglang/zig)

To compile BeatportDL with Zig using Makefile, you must specify the paths to the C/C++ libraries folder and headers folder for the desired OS and architecture with `-L` (for libraries) and `-I` (for headers) flags using environment variables: `MACOS_ARM64_LIB_PATH`, `MACOS_AMD64_LIB_PATH`, `LINUX_AMD64_LIB_PATH`, `LINUX_ARM64_LIB_PATH`, `WINDOWS_AMD64_LIB_PATH`

One line example _(for unix and unix-like os)_

```shell
MACOS_ARM64_LIB_PATH="-L/usr/local/lib -I/usr/local/include" \
make darwin-arm64
```

You can also create an `.env` file in the project folder and specify all environment variables in it:

```
MACOS_ARM64_LIB_PATH=-L/libraries/for/macos-arm64 -I/headers/for/macos-arm64
MACOS_AMD64_LIB_PATH=-L/libraries/for/macos-amd64 -I/headers/for/macos-amd64
LINUX_AMD64_LIB_PATH=-L/libraries/for/linux-amd64 -I/headers/for/linux-amd64
LINUX_ARM64_LIB_PATH=-L/libraries/for/linux-arm64 -I/headers/for/linux-arm64
WINDOWS_AMD64_LIB_PATH=-L/libraries/for/windows-amd64 -I/headers/for/windows-amd64
```
