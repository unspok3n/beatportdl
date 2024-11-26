package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
	"unspok3n/beatportdl/internal/taglib"
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
	fmt.Print("Enter url or search query: ")
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

var (
	ErrUnsupportedLinkType = errors.New("unsupported link type")
)

func (app *application) handleUrl(url string) {
	link, err := app.bp.ParseUrl(url)
	if err != nil {
		LogError("parse url", err)
		return
	}
	switch link.Type {
	default:
		LogError("handle url", ErrUnsupportedLinkType)
	case beatport.TrackLink:
		track, err := app.bp.GetTrack(link.ID)
		if err != nil {
			LogError("fetch track", err)
			return
		}

		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
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
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			LogError("create downloads directory", err)
			return
		}

		location, err := app.saveTrack(*track, downloadsDirectory, app.config.Quality)
		if err != nil {
			LogError("save track", err)
			return
		}
		var coverPath string
		if (app.config.CoverSize != config.DefaultCoverSize && app.config.FixTags) || (app.config.SortByContext && app.config.KeepCover) {
			coverUrl := track.Release.Image.FormattedUrl(app.config.CoverSize)
			coverPath = fmt.Sprintf("%s/%s", downloadsDirectory, uuid.New().String())
			if err = app.downloadFile(coverUrl, coverPath); err != nil {
				LogError("download file", err)
				return
			}
		}

		if err := app.tagTrack(location, *track, coverPath); err != nil {
			LogError("tag track", err)
			return
		}

		if app.config.KeepCover && app.config.SortByContext {
			newPath := filepath.Dir(coverPath) + "/cover.jpg"
			if err := os.Rename(coverPath, newPath); err != nil {
				LogError("rename cover", err)
				return
			}
		} else {
			os.Remove(coverPath)
		}
	case beatport.ReleaseLink:
		release, err := app.bp.GetRelease(link.ID)
		if err != nil {
			LogError("fetch release", err)
			return
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			downloadsDirectory = fmt.Sprintf("%s/%s",
				app.config.DownloadsDirectory,
				release.DirectoryName(
					app.config.ReleaseDirectoryTemplate,
					app.config.WhitespaceCharacter,
				),
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			LogError("create downloads directory", err)
			return
		}

		var coverPath string
		if (app.config.CoverSize != config.DefaultCoverSize && app.config.FixTags) || (app.config.SortByContext && app.config.KeepCover) {
			coverUrl := release.Image.FormattedUrl(app.config.CoverSize)
			coverPath = downloadsDirectory + "/" + uuid.New().String()
			if err = app.downloadFile(coverUrl, coverPath); err != nil {
				LogError("download cover", err)
			}
		}

		wg := sync.WaitGroup{}
		for _, trackUrl := range release.TrackUrls {
			app.backgroundCustom(&wg, func() {
				trackLink, _ := app.bp.ParseUrl(trackUrl)
				track, err := app.bp.GetTrack(trackLink.ID)
				if err != nil {
					LogError("fetch track", err)
					return
				}
				location, err := app.saveTrack(*track, downloadsDirectory, app.config.Quality)
				if err != nil {
					LogError("save track", err)
					return
				}
				if err := app.tagTrack(location, *track, coverPath); err != nil {
					LogError("tag track", err)
					return
				}
			})
		}
		wg.Wait()
		if app.config.KeepCover && app.config.SortByContext {
			newPath := filepath.Dir(coverPath) + "/cover.jpg"
			if err := os.Rename(coverPath, newPath); err != nil {
				LogError("rename cover", err)
				return
			}
		} else {
			os.Remove(coverPath)
		}
	case beatport.PlaylistLink:
		playlist, err := app.bp.GetPlaylist(link.ID)
		if err != nil {
			LogError("fetch playlist", err)
			return
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			downloadsDirectory = fmt.Sprintf("%s/%s",
				app.config.DownloadsDirectory,
				playlist.Name,
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			LogError("create downloads directory", err)
			return
		}

		page := 1
		for {
			items, err := app.bp.GetPlaylistItems(link.ID, page)
			if err != nil {
				LogError("fetch playlist items", err)
				return
			}
			for _, item := range items.Results {
				app.background(func() {
					item.Track.Number = item.Position
					location, err := app.saveTrack(item.Track, downloadsDirectory, app.config.Quality)
					if err != nil {
						LogError("save track", err)
						return
					}
					var coverPath string
					if app.config.CoverSize != config.DefaultCoverSize && app.config.FixTags {
						coverUrl := item.Track.Release.Image.FormattedUrl(app.config.CoverSize)
						coverPath = fmt.Sprintf("%s/%s", downloadsDirectory, uuid.New().String())
						if err = app.downloadFile(coverUrl, coverPath); err != nil {
							LogError("download file", err)
							return
						}
						defer os.Remove(coverPath)
					}

					if err := app.tagTrack(location, item.Track, coverPath); err != nil {
						LogError("tag track", err)
						return
					}
				})
			}

			if items.Next == nil {
				break
			}

			page++
		}
	case beatport.ChartLink:
		chart, err := app.bp.GetChart(link.ID)
		if err != nil {
			LogError("fetch chart", err)
			return
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			downloadsDirectory = fmt.Sprintf("%s/%s",
				app.config.DownloadsDirectory,
				chart.Name,
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			LogError("create downloads directory", err)
			return
		}
		if app.config.KeepCover && app.config.SortByContext {
			app.background(func() {
				coverPath := downloadsDirectory + "/cover.jpg"
				if err = app.downloadFile(chart.Image.FormattedUrl(app.config.CoverSize), coverPath); err != nil {
					LogError("download cover", err)
				}
			})
		}

		page := 1
		for {
			tracks, err := app.bp.GetChartTracks(link.ID, page)
			if err != nil {
				LogError("fetch chart", err)
				return
			}
			for index, track := range tracks.Results {
				app.background(func() {
					track.Number = index + 1
					location, err := app.saveTrack(track, downloadsDirectory, app.config.Quality)
					if err != nil {
						LogError("save track", err)
						return
					}
					var coverPath string
					if app.config.CoverSize != config.DefaultCoverSize && app.config.FixTags {
						coverUrl := track.Release.Image.FormattedUrl(app.config.CoverSize)
						coverPath = fmt.Sprintf("%s/%s", downloadsDirectory, uuid.New().String())
						if err = app.downloadFile(coverUrl, coverPath); err != nil {
							LogError("download file", err)
							return
						}
						defer os.Remove(coverPath)
					}

					if err := app.tagTrack(location, track, coverPath); err != nil {
						LogError("tag track", err)
						return
					}
				})
			}

			if tracks.Next == nil {
				break
			}

			page++
		}
	}
}

func (app *application) saveTrack(track beatport.Track, directory string, quality string) (string, error) {
	stream, err := app.bp.DownloadTrack(track.ID, quality)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("invalid stream quality: %s", stream.StreamQuality)
	}
	fmt.Printf("Downloading %s (%s) [%s]\n", track.Name, track.MixName, displayQuality)
	filePath := fmt.Sprintf("%s/%s%s", directory, fileName, fileExtension)
	if err = app.downloadFile(stream.Location, filePath); err != nil {
		return "", err
	}
	fmt.Printf("Finished downloading %s (%s) [%s]\n", track.Name, track.MixName, displayQuality)

	return filePath, nil
}

var (
	beatportTags = []string{
		"COMMENT",
		"ENCODED_BY",
		"ENCODER",
		"FILEOWNER",
		"FILETYPE",
		"LABEL_URL",
		"INITIAL_KEY",
		"ORGANIZATION",
		"RECORDING_DATE",
		"RELEASE_TIME",
		"TRACK_URL",
		"YEAR",
	}
)

func (app *application) tagTrack(location string, track beatport.Track, coverPath string) error {
	fileExt := filepath.Ext(location)
	if fileExt == ".aac" || !app.config.FixTags {
		return nil
	}
	file, err := taglib.Read(location)
	if err != nil {
		return err
	}
	defer file.Close()

	date := file.GetProperty("RECORDING_DATE")
	key := file.GetProperty("INITIAL_KEY")
	for _, tag := range beatportTags {
		file.SetProperty(tag, "")
	}
	file.SetProperty("DATE", date)
	file.SetProperty("KEY", key)

	if coverPath != "" && app.config.CoverSize != config.DefaultCoverSize {
		data, err := os.ReadFile(coverPath)
		if err != nil {
			return err
		}
		picture := taglib.Picture{
			MimeType:    "image/jpeg",
			PictureType: "Front",
			Description: "Cover",
			Data:        data,
			Size:        uint(len(data)),
		}
		if err := file.SetPicture(&picture); err != nil {
			return err
		}
	}

	if err = file.Save(); err != nil {
		return err
	}

	return nil
}
