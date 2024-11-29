# BeatportDL

Beatport downloader (FLAC, AAC). Supports track and release links.

*Requires [Beatport Streaming Subscription](https://stream.beatport.com/).*

![Screenshot](/screenshots/main.png?raw=true "Screenshot")

Setup
---
[Download](https://github.com/unspok3n/beatportdl/releases/) or [build](#building) beatportDL

*Compiled binaries for Windows, macOS (amd64, arm64) and Linux are available on the [Releases](https://github.com/unspok3n/beatportdl/releases) page*

Run beatportdl, specify username, password, downloads directory, and quality

Subscription types and maximum available quality:\
**Essential** - `medium-hls`*(same as medium on Advanced but uses a slightly slower download method)*\
**Advanced** - `medium`\
**Professional** - `lossless`

*If you use `medium-hls`, you should have [ffmpeg](https://www.ffmpeg.org/download.html) installed*

```shell
./beatportdl
```

This will create a new `beatportdl-config.yml` file

If credentials are correct, you should also see `beatportdl-credentials.json` file appear in beatportdl directory

*If you accidentally typed an incorrect password and got an error, you can always manually edit the config file*

Usage
---

Run beatportdl and enter url or search query
```shell
./beatportdl
```
or specify urls using positional arguments
```shell
./beatportdl https://www.beatport.com/release/slug/12345678 https://www.beatport.com/track/slug/12345678
```

or provide a text file with urls (separated by newline)
```shell
./beatportdl file.txt file2.txt
```

URL types that are currently supported: **Tracks, Releases, Playlists, Charts**

Config options
---
| Option                       | Default                                   | Description                                                                                                       |
|------------------------------|-------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| `username`                   |                                           | Beatport username                                                                                                 |
| `password`                   |                                           | Beatport password                                                                                                 |
| `quality`                    | lossless                                  | Download quality *(medium-hls, medium, high, lossless)*                                                           |
| `downloads_directory`        |                                           | Downloads directory                                                                                               |
| `sort_by_context`            | false                                     | Create a directory for each release, playlist, or chart                                                           |
| `cover_size`                 | 1400x1400                                 | Cover art size for `keep_cover` and track metadata (if `fix_tags` is enabled)  *[max: 1400x1400]*                 |
| `keep_cover`                 | false                                     | Download cover art file (cover.jpg) to the context directory (works only if `sort_by_context` is enabled)         |
| `fix_tags`                   | true                                      | Add missing metadata to M4A (AAC) files and remove useless tags from FLAC files (e.g., Purchased at Beatport.com) |
| `track_file_template`        | {number}. {artists} - {name} ({mix_name}) | Track filename template                                                                                           |
| `release_directory_template` | [{catalog_number}] {artists} - {name}     | Release directory name template                                                                                   |
| `whitespace_character`       |                                           | Whitespace character for track filenames and release directories                                                  |
| `proxy`                      |                                           | Proxy url                                                                                                         |

Download quality:\
`medium-hls` - 128 kbps AAC through the `/stream` endpoint (for Essential subscription, requires [ffmpeg](https://www.ffmpeg.org/download.html))\
`medium` - 128 kbps AAC\
`high` - 256 kbps AAC\
`lossless` - 44.1 khz FLAC

Available template keywords:
* Track: `id`,`name`,`mix_name`,`artists`,`remixers`,`number`,`key`,`bpm`,`genre`,`isrc`
* Release: `id`,`name`,`artists`,`remixers`,`date`,`catalog_number`

Proxy url example: `http://username:password@127.0.0.1:8080`

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