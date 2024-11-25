package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
)

func Setup() (cfg *config.AppConfig, cachePath string) {
	configFilePath, err := FindFile(configFilename)
	if err != nil {
		fmt.Println("Config file not found, creating a new one")
		configFilePath, err = ExecutableDirFilePath(configFilename)
		if err != nil {
			FatalError("get executable path", err)
		}

		fmt.Print("Username: ")
		username := GetLine()
		fmt.Print("Password: ")
		password := GetLine()
		fmt.Print("Downloads directory: ")
		downloadsDir := GetLine()

		cfg := &config.AppConfig{
			Username:           username,
			Password:           password,
			DownloadsDirectory: downloadsDir,
		}
		if err := cfg.Save(configFilePath); err != nil {
			FatalError("save config", err)
		}
	}

	parsedConfig, err := config.Parse(configFilePath)
	if err != nil {
		FatalError("load config", err)
	}

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

	return parsedConfig, cacheFilePath
}

func (app *application) mainPrompt() {
	fmt.Print("Enter track or release link or search query: ")
	input := GetLine()
	if strings.HasPrefix(input, "https://www.beatport.com") {
		app.urls = append(app.urls, input)
	} else {
		app.search(input)
	}
}

func (app *application) search(input string) {
	results, err := app.bp.Search(input)
	if err != nil {
		FatalError("beatport", err)
	}
	trackResultsLen := len(results.Tracks)
	releasesResultsLen := len(results.Releases)

	if trackResultsLen+releasesResultsLen == 0 {
		fmt.Println("No results found")
		return
	}

	fmt.Println("Search results:")
	fmt.Println("[ Tracks ]")
	for i, track := range results.Tracks {
		fmt.Printf(
			"%2d. %s - %s (%s) [%s]\n", i+1,
			track.ArtistsDisplay(beatport.ArtistTypeMain),
			track.Name,
			track.MixName,
			track.Length,
		)
	}
	fmt.Println("\n[ Releases ]")
	indexOffset := trackResultsLen + 1
	for i, release := range results.Releases {
		fmt.Printf(
			"%2d. %s - %s [%s]\n", i+indexOffset,
			release.ArtistsDisplay(beatport.ArtistTypeMain),
			release.Name,
			release.Label.Name,
		)
	}
	fmt.Print("Enter the result number(s): ")
	input = GetLine()
	requestedResults := strings.Split(input, " ")
	for _, result := range requestedResults {
		resultInt, err := strconv.Atoi(result)
		if err != nil {
			fmt.Printf("invalid result number: %s\n", result)
			continue
		}

		if resultInt > releasesResultsLen+trackResultsLen || resultInt == 0 {
			fmt.Printf("invalid result number: %d\n", resultInt)
			continue
		}

		if resultInt >= indexOffset {
			app.urls = append(app.urls, results.Releases[resultInt-indexOffset].URL)
		} else {
			app.urls = append(app.urls, results.Tracks[resultInt-1].URL)
		}
	}
}

func (app *application) parseTextFile(path string) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		FatalError("read input text file", err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		app.urls = append(app.urls, scanner.Text())
	}
}

func (app *application) handleUrl(url string) {
	link, err := app.bp.ParseUrl(url)
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

		var coverPath string
		var coverUrl string

		if app.config.CreateReleaseDirectory {
			release, err := app.bp.GetRelease(track.Release.ID)
			if err != nil {
				LogError("fetch release", err)
				return
			}
			releaseDirectory := release.DirectoryName(
				app.config.ReleaseDirectoryTemplate,
				app.config.WhitespaceCharacter,
			)
			downloadsDirectory = fmt.Sprintf("%s/%s",
				downloadsDirectory,
				releaseDirectory,
			)
			if app.config.CoverSize != "" {
				coverUrl = strings.Replace(
					release.Image.DynamicURI,
					"{w}x{h}",
					app.config.CoverSize,
					-1,
				)
				coverPath = downloadsDirectory + "/cover.jpg"
			}
		}

		if err := CreateDirectory(downloadsDirectory); err != nil {
			LogError("create downloads directory", err)
			return
		}

		if coverUrl != "" && coverPath != "" {
			if err = app.downloadFile(coverUrl, coverPath); err != nil {
				LogError("download cover", err)
			}
		}

		if err := app.saveTrack(*track, downloadsDirectory, app.config.Quality); err != nil {
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
			releaseDirectory := release.DirectoryName(
				app.config.ReleaseDirectoryTemplate,
				app.config.WhitespaceCharacter,
			)
			downloadsDirectory = fmt.Sprintf("%s/%s",
				app.config.DownloadsDirectory,
				releaseDirectory,
			)
		}

		if err := CreateDirectory(downloadsDirectory); err != nil {
			LogError("create downloads directory", err)
			return
		}

		if app.config.CoverSize != "" {
			coverUrl := strings.Replace(
				release.Image.DynamicURI,
				"{w}x{h}",
				app.config.CoverSize,
				-1,
			)
			coverPath := downloadsDirectory + "/cover.jpg"
			if err = app.downloadFile(coverUrl, coverPath); err != nil {
				LogError("download cover", err)
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
				if err := app.saveTrack(*track, downloadsDirectory, app.config.Quality); err != nil {
					LogError("save track", err)
					return
				}
			})
		}
	}
}

func (app *application) saveTrack(track beatport.Track, directory string, quality string) error {
	stream, err := app.bp.DownloadTrack(track.ID, quality)
	if err != nil {
		return err
	}
	fileName := track.Filename(app.config.TrackFileTemplate, app.config.WhitespaceCharacter)
	var fileExtension string
	var displayQuality string
	switch stream.StreamQuality {
	case ".128k.aac.mp4":
		fileExtension = ".aac"
		displayQuality = "AAC 128kbps"
	case ".256k.aac.mp4":
		fileExtension = ".aac"
		displayQuality = "AAC 256kbps"
	case ".flac":
		fileExtension = ".flac"
		displayQuality = "FLAC"
	default:
		return fmt.Errorf("invalid stream quality: %s", stream.StreamQuality)
	}
	fmt.Printf("Downloading %s (%s) [%s]\n", track.Name, track.MixName, displayQuality)
	filePath := fmt.Sprintf("%s/%s%s", directory, fileName, fileExtension)
	if err = app.downloadFile(stream.Location, filePath); err != nil {
		return err
	}
	fmt.Printf("Finished downloading %s (%s) [%s]\n", track.Name, track.MixName, displayQuality)

	return nil
}
