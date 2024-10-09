# BeatportDL

Beatport FLAC downloader. Supports track and release links.

*Requires [Beatport Professional Subscription](https://stream.beatport.com/).*

![Screenshot](/screenshots/main.png?raw=true "Screenshot")

Setup
---
Download or build beatportDL

*Compiled binaries for Windows, macOS (amd64, arm64) and Linux are available on the [Releases](https://github.com/unspok3n/beatportdl/releases) page*

Create `beatportdl-config.yml` file and specify the desired downloads directory

Example:
```yml
downloads_directory: '/users/name/downloads/beatportdl'
```

Run beatportdl with `--authorize` flag
```shell
./beatportdl --authorize
```

Open the OAuth URL in your browser and wait for the redirect

*You may be prompted to login to your Beatport account*

Copy the `code` value from the address bar

![Screenshot](/screenshots/code.png?raw=true "Screenshot")

![Screenshot](/screenshots/authorize.png?raw=true "Screenshot")

If everything went well, you should see the `beatportdl-credentials.json` file appear in the beatportdl directory.

*You may need to repeat this process if you haven't used the downloader for some time and the credentials have expired. ("Authentication credentials were not provided" error)*

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

Config options
---
| Option                       | Default                                   | Description                                                      |
|------------------------------|-------------------------------------------|------------------------------------------------------------------|
| `create_release_directory`   | false                                     | Create directory per release                                     |
| `track_file_template`        | {number}. {artists} - {name} ({mix_name}) | Track filename template                                          |
| `release_directory_template` | [{catalog_number}] {artists} - {name}     | Release directory name template                                  |
| `whitespace_character`       |                                           | Whitespace character for track filenames and release directories |
| `proxy`                      |                                           | Proxy url                                                        |

Available template keywords:
* Track: `id`,`name`,`mix_name`,`artists`,`remixers`,`number`,`key`,`bpm`,`genre`,`isrc`
* Release: `id`,`name`,`artists`,`remixers`,`date`,`catalog_number`