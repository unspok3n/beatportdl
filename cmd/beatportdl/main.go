package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
)

const (
	configFilename = "beatportdl-config.yml"
	cacheFilename  = "beatportdl-credentials.json"
)

type application struct {
	config *config.AppConfig
	bp     *beatport.Beatport
	wg     sync.WaitGroup
}

func main() {
	configFilePath, err := FindFile(configFilename)
	if err != nil {
		FatalError("find config file", err)
	}
	parsedConfig, err := config.ParseConfig(configFilePath)
	if err != nil {
		FatalError("load config", err)
	}

	flag.Parse()
	urls := flag.Args()

	execCachePath, err := ExecutableDirFilePath(cacheFilename)
	if err != nil {
		FatalError("get executable path", err)
	}
	cacheFilePath := execCachePath

	_, err = os.Stat(cacheFilePath)
	if err != nil {
		workingCachePath, err := WorkingDirFilePath(cacheFilename)
		if err != nil {
			FatalError("get current working dir", err)
		}
		_, err = os.Stat(workingCachePath)
		if err == nil {
			cacheFilePath = workingCachePath
		}
	}

	bpClient, err := beatport.New(
		parsedConfig.Username,
		parsedConfig.Password,
		cacheFilePath,
		parsedConfig.Proxy,
	)
	if err != nil {
		FatalError("beatport api client", err)
	}

	if err := bpClient.LoadCachedTokenPair(); err != nil {
		fmt.Println("Authorizing")
		if err := bpClient.Authorize(); err != nil {
			FatalError("beatport", err)
		}
	}

	app := &application{
		config: parsedConfig,
		bp:     bpClient,
	}

	if len(urls) == 0 {
		fmt.Print("Enter track or release link: ")
		input := GetLine()
		urls = append(urls, input)
	}

	for _, input := range urls {
		app.background(func() {
			link, err := app.bp.ParseUrl(input)
			if err != nil {
				LogError("parse url", err)
				return
			}
			if link.Type == beatport.TrackLink {
				downloadsDirectory := app.config.DownloadsDirectory
				track, err := app.bp.GetTrack(link.ID)
				if err != nil {
					LogError("fetch track", err)
					return
				}

				if app.config.CreateReleaseDirectory {
					release, err := app.bp.GetRelease(track.Release.ID)
					if err != nil {
						LogError("fetch release", err)
						return
					}

					releaseDirectory := release.DirectoryName(app.config.ReleaseDirectoryTemplate)
					if app.config.WhitespaceCharacter != " " {
						releaseDirectory = strings.Replace(
							releaseDirectory,
							" ",
							app.config.WhitespaceCharacter,
							-1,
						)
					}
					downloadsDirectory = fmt.Sprintf("%s/%s",
						downloadsDirectory,
						releaseDirectory,
					)

				}
				if _, err := os.Stat(downloadsDirectory); os.IsNotExist(err) {
					if err := os.MkdirAll(downloadsDirectory, 0760); err != nil {
						LogError("create downloads directory", err)
						return
					}
				}
				if err := app.saveTrack(*track, downloadsDirectory); err != nil {
					LogError("save track", err)
					return
				}
			} else if link.Type == beatport.ReleaseLink {
				release, err := app.bp.GetRelease(link.ID)
				if err != nil {
					LogError("fetch release", err)
					return
				}
				downloadsDirectory := app.config.DownloadsDirectory
				if app.config.CreateReleaseDirectory {
					releaseDirectory := release.DirectoryName(app.config.ReleaseDirectoryTemplate)
					if app.config.WhitespaceCharacter != " " {
						releaseDirectory = strings.Replace(
							releaseDirectory,
							" ",
							app.config.WhitespaceCharacter,
							-1,
						)
					}
					downloadsDirectory = fmt.Sprintf("%s/%s",
						app.config.DownloadsDirectory,
						releaseDirectory,
					)
				}

				if _, err := os.Stat(downloadsDirectory); os.IsNotExist(err) {
					if err := os.MkdirAll(downloadsDirectory, 0760); err != nil {
						LogError("create downloads directory", err)
						return
					}
				}

				for _, trackUrl := range release.TrackUrls {
					app.background(func() {
						trackLink, _ := app.bp.ParseUrl(trackUrl)
						track, err := app.bp.GetTrack(trackLink.ID)
						if err != nil {
							LogError("fetch track", err)
							return
						}
						if err := app.saveTrack(*track, downloadsDirectory); err != nil {
							LogError("save track", err)
							return
						}
					})
				}
			}
		})
	}

	app.wg.Wait()

	Pause()
}

func (app *application) saveTrack(track beatport.Track, directory string) error {
	fmt.Printf("Downloading %s (%s)\n", track.Name, track.MixName)
	stream, err := app.bp.DownloadTrack(track.ID)
	if err != nil {
		return err
	}
	fileName := track.Filename(app.config.TrackFileTemplate)
	if app.config.WhitespaceCharacter != " " {
		fileName = strings.Replace(fileName, " ", app.config.WhitespaceCharacter, -1)
	}
	filePath := fmt.Sprintf("%s/%s", directory, fileName)
	if err = app.downloadFile(stream.Location, filePath); err != nil {
		return err
	}
	fmt.Printf("Finished downloading %s (%s)\n", track.Name, track.MixName)

	return nil
}
