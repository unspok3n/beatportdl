# BeatportDL

Beatport downloader (FLAC, AAC). Supports `track` and `release` links.

*Requires a [Beatport Streaming Subscription](https://stream.beatport.com/).*

![Screenshot](/screenshots/main.png?raw=true "Screenshot")

Setup
---
1. [Download](https://github.com/unspok3n/beatportdl/releases/) or [build](#building) BeatportDL.

     *Compiled binaries for Windows, macOS (amd64, arm64) and Linux are available on the [Releases](https://github.com/unspok3n/beatportdl/releases) page.*

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
| Option                       | Default Value                             | Type    | Description                                                                                                       |
|------------------------------|-------------------------------------------|---------|-------------------------------------------------------------------------------------------------------------------|
| `username`                   |                                           | String  | Beatport username                                                                                                 |
| `password`                   |                                           | String  | Beatport password                                                                                                 |
| `quality`                    | lossless                                  | String  | Download quality *(medium-hls, medium, high, lossless)*                                                           |
| `downloads_directory`        |                                           | String  | Location for the downloads directory                                                                              |
| `sort_by_context`            | false                                     | Boolean | Create a directory for each release, playlist, or chart                                                           |
| `cover_size`                 | 1400x1400                                 | String  | Cover art size for `keep_cover` and track metadata (if `fix_tags` is enabled)  *[max: 1400x1400]*                 |
| `keep_cover`                 | false                                     | Boolean | Download cover art file (cover.jpg) to the context directory (works only if `sort_by_context` is enabled)         |
| `fix_tags`                   | true                                      | Boolean | Add missing metadata to M4A (AAC) files and remove useless tags from FLAC files (e.g., Purchased at Beatport.com) |
| `track_file_template`        | {number}. {artists} - {name} ({mix_name}) | String  | Track filename template                                                                                           |
| `release_directory_template` | [{catalog_number}] {artists} - {name}     | String  | Release directory name template                                                                                   |
| `whitespace_character`       |                                           | String  | Whitespace character for track filenames and release directories                                                  |
| `proxy`                      |                                           | String  | Proxy URL                                                                                                         |

Download quality options, per Beatport subscription type:

| Option       | Description                                                                                                                  | Requires at least | Notes                                                                   |
|--------------|------------------------------------------------------------------------------------------------------------------------------|-------------------|-------------------------------------------------------------------------|
| `medium-hls` | 128 kbps AAC through `/stream` endpoint (IMPORTANT: requires [ffmpeg](https://www.ffmpeg.org/download.html) to be installed) | Essential         | Same as `medium` on Advanced but uses a slightly slower download method |
| `medium`     | 128 kbps AAC                                                                                                                 | Advanced          |                                                                         |
| `high`       | 256 kbps AAC                                                                                                                 | Professional      |                                                                         |
| `lossless`   | 44.1 kHz FLAC                                                                                                                | Professional      |                                                                         |

If the Beatport credentials are correct, you should also see the file `beatportdl-credentials.json` appear in the BeatportDL directory.
*If you accidentally entered an incorrect password and got an error, you can always manually edit the config file*

Available template keywords for `track_file_template` & `release_directory_template`:
* Track: `id`,`name`,`mix_name`,`artists`,`remixers`,`number`,`key`,`bpm`,`genre`,`isrc`
* Release: `id`,`name`,`artists`,`remixers`,`date`,`catalog_number`

Proxy URL format example: `http://username:password@127.0.0.1:8080`

Usage
---

Run BeatportDL and enter the Beatport URL or search query:
```shell
./beatportdl
Enter url or search query:
```
...or specify the URL using positional arguments:
```shell
./beatportdl https://www.beatport.com/track/strobe/1696999 https://www.beatport.com/track/move-for-me/591753
```
...or provide a text file with urls (separated by a newline)
```shell
./beatportdl file.txt file2.txt
```

URL types that are currently supported: **Tracks, Releases, Playlists, Charts**
(Labels & Artist coming soon!)

Building
---
Required dependencies:
* [TagLib](https://github.com/taglib/taglib) >= 2.0
* [zlib](https://github.com/madler/zlib) >= 1.2.3
* [Zig C/C++ Toolchain](https://github.com/ziglang/zig) >= 0.13.0

BeatportDL uses [TagLib](https://taglib.org/) C bindings to handle audio metadata and therefore requires `CGO_ENABLED=1`

Makefile is adapted for CGO cross-compilation and uses [Zig toolchain](https://github.com/ziglang/zig)

To compile BeatportDL with Zig using Makefile, you have to specify the path to C/C++ libraries and headers for the desired OS and architecture using environment variables:
```shell
MACOS_ARM64_LIB_PATH=
MACOS_AMD64_LIB_PATH=
LINUX_AMD64_LIB_PATH=
WINDOWS_AMD64_LIB_PATH=
```
Example
```shell
MACOS_ARM64_LIB_PATH="-L/usr/lib/aarch64-macos -I/usr/include/aarch64-macos" \
make darwin-arm64
```
