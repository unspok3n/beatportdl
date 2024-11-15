# BeatportDL

Beatport downloader (FLAC, AAC). Supports track and release links.

*Requires [Beatport Streaming Subscription](https://stream.beatport.com/).*

![Screenshot](/screenshots/main.png?raw=true "Screenshot")

Setup
---
Download or build beatportDL

*Compiled binaries for Windows, macOS (amd64, arm64) and Linux are available on the [Releases](https://github.com/unspok3n/beatportdl/releases) page*

Run beatportdl, specify username, password, and downloads directory
```shell
./beatportdl
```
This will create a new `beatportdl-config.yml` file

If credentials are correct, you should also see `beatportdl-credentials.json` file appear in beatportdl directory

*If you accidentally typed an incorrect password and got an error, you can always manually edit the config file*

Usage
---

Run beatportdl and enter the track or release url
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

Config options
---
| Option                       | Default                                   | Description                                                      |
|------------------------------|-------------------------------------------|------------------------------------------------------------------|
| `username`                   |                                           | Beatport username                                                |
| `password`                   |                                           | Beatport password                                                |
| `quality`                    | lossless                                  | Download quality *(medium, high, lossless)*                      |
| `downloads_directory`        |                                           | Downloads directory                                              |
| `create_release_directory`   | false                                     | Create directory per release                                     |
| `cover_size`                 |                                           | Cover art size *(max: 1400x1400)*                                |
| `track_file_template`        | {number}. {artists} - {name} ({mix_name}) | Track filename template                                          |
| `release_directory_template` | [{catalog_number}] {artists} - {name}     | Release directory name template                                  |
| `whitespace_character`       |                                           | Whitespace character for track filenames and release directories |
| `proxy`                      |                                           | Proxy url                                                        |

Download quality:\
`medium` - 128 kbps AAC\
`high` - 256 kbps AAC\
`lossless` - 44.1 khz FLAC

Available template keywords:
* Track: `id`,`name`,`mix_name`,`artists`,`remixers`,`number`,`key`,`bpm`,`genre`,`isrc`
* Release: `id`,`name`,`artists`,`remixers`,`date`,`catalog_number`

Proxy url example: `http://username:password@127.0.0.1:8080`