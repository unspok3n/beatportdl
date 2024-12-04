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

func Setup() (cfg *config.AppConfig, cachePath string, err error) {
	configFilePath, err := FindFile(configFilename)
	if err != nil {
		fmt.Println("Config file not found, creating a new one")
		configFilePath, err = ExecutableDirFilePath(configFilename)
		if err != nil {
			return nil, configFilePath, fmt.Errorf("get executable path: %w", err)
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

		fmt.Println("1. Lossless (44.1 khz FLAC)\n2. High (256 kbps AAC)\n3. Medium (128 kbps AAC)\n4. Medium HLS (128 kbps AAC)")
		for {
			fmt.Print("Quality: ")
			qualityNumber := GetLine()
			switch qualityNumber {
			case "1":
				cfg.Quality = "lossless"
			case "2":
				cfg.Quality = "high"
			case "3":
				cfg.Quality = "medium"
			case "4":
				cfg.Quality = "medium-hls"
			default:
				fmt.Println("Invalid quality")
				continue
			}
			break
		}

		if err := cfg.Save(configFilePath); err != nil {
			return nil, configFilePath, fmt.Errorf("save config: %w", err)
		}
	}

	parsedConfig, err := config.Parse(configFilePath)
	if err != nil {
		return nil, configFilePath, fmt.Errorf("load config: %w", err)
	}

	execCachePath, err := ExecutableDirFilePath(cacheFilename)
	if err != nil {
		return nil, configFilePath, fmt.Errorf("get executable path: %w", err)
	}
	cacheFilePath := execCachePath

	_, err = os.Stat(cacheFilePath)
	if err != nil {
		workingCachePath, err := WorkingDirFilePath(cacheFilename)
		if err != nil {
			return nil, configFilePath, fmt.Errorf("get working dir path: %w", err)
		}
		_, err = os.Stat(workingCachePath)
		if err == nil {
			cacheFilePath = workingCachePath
		}
	}

	return parsedConfig, cacheFilePath, nil
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
		app.FatalError("beatport", err)
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
			track.ArtistsDisplay(
				beatport.ArtistTypeMain,
				app.config.ArtistsLimit,
				app.config.ArtistsShortForm,
			),
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
			release.ArtistsDisplay(
				beatport.ArtistTypeMain,
				app.config.ArtistsLimit,
				app.config.ArtistsShortForm,
			),
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
		app.FatalError("read input text file", err)
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
		app.LogError(fmt.Sprintf("[%s] parse url", url), err)
		return
	}
	switch link.Type {
	default:
		app.LogError("handle url", ErrUnsupportedLinkType)
	case beatport.TrackLink:
		track, err := app.bp.GetTrack(link.ID)
		if err != nil {
			app.LogError(fmt.Sprintf("[%s] fetch track", url), err)
			return
		}

		downloadsDirectory := app.config.DownloadsDirectory
		release, err := app.bp.GetRelease(track.Release.ID)
		if err != nil {
			app.LogError(fmt.Sprintf("[%s] fetch track release", url), err)
			return
		}
		if app.config.SortByContext {
			releaseDirectory := release.DirectoryName(
				app.config.ReleaseDirectoryTemplate,
				app.config.WhitespaceCharacter,
				app.config.ArtistsLimit,
				app.config.ArtistsShortForm,
			)
			if app.config.CreateLabelDirectory {
				downloadsDirectory = fmt.Sprintf("%s/%s",
					downloadsDirectory,
					release.Label.NameSanitized(),
				)
			}
			downloadsDirectory = fmt.Sprintf("%s/%s",
				downloadsDirectory,
				releaseDirectory,
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			app.LogError(fmt.Sprintf("[%s] create downloads directory", url), err)
			return
		}
		track.Release = *release
		app.withSemaphore(func() {
			location, err := app.saveTrack(*track, downloadsDirectory, app.config.Quality)
			if err != nil {
				app.LogError(fmt.Sprintf("[%s] save track", url), err)
				return
			}
			var coverPath string
			if app.requireCover(true) {
				coverUrl := track.Release.Image.FormattedUrl(app.config.CoverSize)
				coverPath = fmt.Sprintf("%s/%s", downloadsDirectory, uuid.New().String())
				if err = app.downloadFile(coverUrl, coverPath); err != nil {
					app.LogError(fmt.Sprintf("[%s] download file", url), err)
					return
				}
			}

			if err := app.tagTrack(location, *track, coverPath); err != nil {
				app.LogError(fmt.Sprintf("[%s] tag track", url), err)
				return
			}

			if app.config.KeepCover && app.config.SortByContext {
				newPath := filepath.Dir(coverPath) + "/cover.jpg"
				if err := os.Rename(coverPath, newPath); err != nil {
					app.LogError(fmt.Sprintf("[%s] rename cover", url), err)
					return
				}
			} else {
				os.Remove(coverPath)
			}
		})
	case beatport.ReleaseLink:
		release, err := app.bp.GetRelease(link.ID)
		if err != nil {
			app.LogError(fmt.Sprintf("[%s] fetch release", url), err)
			return
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			if app.config.CreateLabelDirectory {
				downloadsDirectory = fmt.Sprintf("%s/%s",
					downloadsDirectory,
					release.Label.NameSanitized(),
				)
			}
			downloadsDirectory = fmt.Sprintf("%s/%s",
				downloadsDirectory,
				release.DirectoryName(
					app.config.ReleaseDirectoryTemplate,
					app.config.WhitespaceCharacter,
					app.config.ArtistsLimit,
					app.config.ArtistsShortForm,
				),
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			app.LogError(fmt.Sprintf("[%s] create downloads directory", url), err)
			return
		}

		var coverPath string
		if app.requireCover(true) {
			coverUrl := release.Image.FormattedUrl(app.config.CoverSize)
			coverPath = downloadsDirectory + "/" + uuid.New().String()
			if err = app.downloadFile(coverUrl, coverPath); err != nil {
				app.LogError(fmt.Sprintf("[%s] download cover", url), err)
			}
		}

		wg := sync.WaitGroup{}
		for _, trackUrl := range release.TrackUrls {
			app.withSemaphoreCustom(&wg, func() {
				trackLink, _ := app.bp.ParseUrl(trackUrl)
				track, err := app.bp.GetTrack(trackLink.ID)
				if err != nil {
					app.LogError(fmt.Sprintf("[%s] fetch track '%d'", url, trackLink.ID), err)
					return
				}
				track.Release = *release
				location, err := app.saveTrack(*track, downloadsDirectory, app.config.Quality)
				if err != nil {
					app.LogError(fmt.Sprintf("[%s] save track '%d'", url, trackLink.ID), err)
					return
				}
				if err := app.tagTrack(location, *track, coverPath); err != nil {
					app.LogError(fmt.Sprintf("[%s] tag track '%d'", url, trackLink.ID), err)
					return
				}
			})
		}
		wg.Wait()
		if app.config.KeepCover && app.config.SortByContext {
			newPath := filepath.Dir(coverPath) + "/cover.jpg"
			if err := os.Rename(coverPath, newPath); err != nil {
				app.LogError(fmt.Sprintf("[%s] rename cover", url), err)
				return
			}
		} else {
			os.Remove(coverPath)
		}
	case beatport.PlaylistLink:
		playlist, err := app.bp.GetPlaylist(link.ID)
		if err != nil {
			app.LogError(fmt.Sprintf("[%s] fetch playlist", url), err)
			return
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			downloadsDirectory = fmt.Sprintf("%s/%s",
				downloadsDirectory,
				playlist.Name,
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			app.LogError(fmt.Sprintf("[%s] create downloads directory", url), err)
			return
		}

		page := 1
		for {
			items, err := app.bp.GetPlaylistItems(link.ID, page)
			if err != nil {
				app.LogError(fmt.Sprintf("[%s] fetch playlist items", url), err)
				return
			}
			for _, item := range items.Results {
				app.withSemaphore(func() {
					item.Track.Number = item.Position
					release, err := app.bp.GetRelease(item.Track.Release.ID)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] fetch track release '%d'", url, item.Track.ID), err)
						return
					}
					item.Track.Release = *release
					location, err := app.saveTrack(item.Track, downloadsDirectory, app.config.Quality)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] save track '%d'", url, item.Track.ID), err)
						return
					}
					var coverPath string
					if app.requireCover(false) {
						coverUrl := item.Track.Release.Image.FormattedUrl(app.config.CoverSize)
						coverPath = fmt.Sprintf("%s/%s", downloadsDirectory, uuid.New().String())
						if err = app.downloadFile(coverUrl, coverPath); err != nil {
							app.LogError(fmt.Sprintf("[%s] download track cover '%d'", url, item.Track.ID), err)
							return
						}
						defer os.Remove(coverPath)
					}

					if err := app.tagTrack(location, item.Track, coverPath); err != nil {
						app.LogError(fmt.Sprintf("[%s] tag track '%d'", url, item.Track.ID), err)
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
			app.LogError(fmt.Sprintf("[%s] fetch chart", url), err)
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
			app.LogError(fmt.Sprintf("[%s] create downloads directory", url), err)
			return
		}
		if app.config.KeepCover && app.config.SortByContext {
			app.withSemaphore(func() {
				coverPath := downloadsDirectory + "/cover.jpg"
				if err = app.downloadFile(chart.Image.FormattedUrl(app.config.CoverSize), coverPath); err != nil {
					app.LogError(fmt.Sprintf("[%s] download chart cover", url), err)
				}
			})
		}

		page := 1
		for {
			tracks, err := app.bp.GetChartTracks(link.ID, page)
			if err != nil {
				app.LogError(fmt.Sprintf("[%s] fetch chart tracks", url), err)
				return
			}
			for index, track := range tracks.Results {
				app.withSemaphore(func() {
					track.Number = index + 1
					release, err := app.bp.GetRelease(track.Release.ID)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] fetch track release '%d'", url, track.ID), err)
						return
					}
					track.Release = *release
					location, err := app.saveTrack(track, downloadsDirectory, app.config.Quality)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] save track '%d'", url, track.ID), err)
						return
					}
					var coverPath string
					if app.requireCover(false) {
						coverUrl := track.Release.Image.FormattedUrl(app.config.CoverSize)
						coverPath = fmt.Sprintf("%s/%s", downloadsDirectory, uuid.New().String())
						if err = app.downloadFile(coverUrl, coverPath); err != nil {
							app.LogError(fmt.Sprintf("[%s] save track '%d'", url, track.ID), err)
							return
						}
						defer os.Remove(coverPath)
					}

					if err := app.tagTrack(location, track, coverPath); err != nil {
						app.LogError(fmt.Sprintf("[%s] tag track '%d'", url, track.ID), err)
						return
					}
				})
			}

			if tracks.Next == nil {
				break
			}

			page++
		}
	case beatport.LabelLink:
		label, err := app.bp.GetLabel(link.ID)
		if err != nil {
			app.LogError(fmt.Sprintf("[%s] fetch label", url), err)
			return
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			downloadsDirectory = fmt.Sprintf("%s/%s",
				app.config.DownloadsDirectory,
				label.NameSanitized(),
			)
		}
		if err := CreateDirectory(downloadsDirectory); err != nil {
			app.LogError(fmt.Sprintf("[%s] create downloads directory", url), err)
			return
		}

		page := 1
		for {
			releases, err := app.bp.GetLabelReleases(link.ID, page)
			if err != nil {
				app.LogError(fmt.Sprintf("[%s] fetch label releases", url), err)
				return
			}
			for _, release := range releases.Results {
				app.background(func() {
					releaseDirectory := downloadsDirectory
					if app.config.SortByContext {
						releaseDirectory = fmt.Sprintf("%s/%s",
							releaseDirectory,
							release.DirectoryName(
								app.config.ReleaseDirectoryTemplate,
								app.config.WhitespaceCharacter,
								app.config.ArtistsLimit,
								app.config.ArtistsShortForm,
							),
						)
						if err := CreateDirectory(releaseDirectory); err != nil {
							app.LogError(fmt.Sprintf("[%s] create downloads directory for release '%d'", url, release.ID), err)
							return
						}
					}

					var coverPath string
					if app.requireCover(true) {
						coverUrl := release.Image.FormattedUrl(app.config.CoverSize)
						coverPath = releaseDirectory + "/" + uuid.New().String()
						app.withSemaphore(func() {
							if err = app.downloadFile(coverUrl, coverPath); err != nil {
								app.LogError(fmt.Sprintf("[%s] download cover for release '%d'", url, release.ID), err)
							}
						})
					}

					tPage := 1
					for {
						tracks, err := app.bp.GetReleaseTracks(release.ID, tPage)
						if err != nil {
							app.LogError(fmt.Sprintf("[%s] fetch label release tracks '%d'", url, release.ID), err)
							return
						}

						for _, track := range tracks.Results {
							app.withSemaphore(func() {
								trackFull, err := app.bp.GetTrack(track.ID)
								if err != nil {
									app.LogError(fmt.Sprintf("[%s] fetch track '%d'", url, track.ID), err)
									return
								}
								trackFull.Release = release
								location, err := app.saveTrack(*trackFull, releaseDirectory, app.config.Quality)
								if err != nil {
									app.LogError(fmt.Sprintf("[%s] save track '%d'", url, trackFull.ID), err)
									return
								}
								if err := app.tagTrack(location, *trackFull, coverPath); err != nil {
									app.LogError(fmt.Sprintf("[%s] tag track '%d'", url, trackFull.ID), err)
									return
								}
							})
						}

						if tracks.Next == nil {
							break
						}
						tPage++
					}

					if app.config.KeepCover && app.config.SortByContext {
						newPath := filepath.Dir(coverPath) + "/cover.jpg"
						if err := os.Rename(coverPath, newPath); err != nil {
							app.LogError(fmt.Sprintf("[%s] rename cover", url), err)
							return
						}
					} else {
						os.Remove(coverPath)
					}
				})
			}

			if releases.Next == nil {
				break
			}

			page++
		}
	case beatport.ArtistLink:
		artist, err := app.bp.GetArtist(link.ID)
		if err != nil {
			app.LogError(fmt.Sprintf("[%s] fetch artist", url), err)
		}
		downloadsDirectory := app.config.DownloadsDirectory
		if app.config.SortByContext {
			downloadsDirectory = fmt.Sprintf("%s/%s", downloadsDirectory, artist.NameSanitized())
		}

		if err := CreateDirectory(downloadsDirectory); err != nil {
			app.LogError(fmt.Sprintf("[%s] create downloads directory", url), err)
			return
		}

		page := 1
		for {
			tracks, err := app.bp.GetArtistTracks(link.ID, page)
			if err != nil {
				app.LogError(fmt.Sprintf("[%s] fetch artist tracks", url), err)
				return
			}
			for _, track := range tracks.Results {
				app.withSemaphore(func() {
					trackFull, err := app.bp.GetTrack(track.ID)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] fetch track '%d'", url, track.ID), err)
						return
					}
					releaseDownloadsDirectory := downloadsDirectory

					release, err := app.bp.GetRelease(track.Release.ID)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] fetch track release '%d'", url, track.ID), err)
						return
					}
					track.Release = *release
					if app.config.SortByContext {
						releaseDownloadsDirectory = fmt.Sprintf("%s/%s",
							releaseDownloadsDirectory,
							release.DirectoryName(
								app.config.ReleaseDirectoryTemplate,
								app.config.WhitespaceCharacter,
								app.config.ArtistsLimit,
								app.config.ArtistsShortForm,
							),
						)
						if err := CreateDirectory(releaseDownloadsDirectory); err != nil {
							app.LogError(fmt.Sprintf("[%s] create downloads directory for track '%d'", url, track.ID), err)
							return
						}
					}

					var coverPath string
					if app.requireCover(true) {
						coverUrl := track.Release.Image.FormattedUrl(app.config.CoverSize)
						coverPath = releaseDownloadsDirectory + "/" + uuid.New().String()
						if err = app.downloadFile(coverUrl, coverPath); err != nil {
							app.LogError(fmt.Sprintf("[%s] download cover", url), err)
						}
					}

					location, err := app.saveTrack(*trackFull, releaseDownloadsDirectory, app.config.Quality)
					if err != nil {
						app.LogError(fmt.Sprintf("[%s] save track '%d'", url, track.ID), err)
						return
					}
					if err := app.tagTrack(location, *trackFull, coverPath); err != nil {
						app.LogError(fmt.Sprintf("[%s] tag track '%d'", url, track.ID), err)
						return
					}

					if app.config.KeepCover && app.config.SortByContext {
						newPath := filepath.Dir(coverPath) + "/cover.jpg"
						if err := os.Rename(coverPath, newPath); err != nil {
							app.LogError(fmt.Sprintf("[%s] rename cover", url), err)
							return
						}
					} else {
						os.Remove(coverPath)
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

func (app *application) requireCover(respectKeepCover bool) bool {
	fixTags := (app.config.CoverSize != config.DefaultCoverSize ||
		app.config.Quality != "lossless") && app.config.FixTags
	keepCover := app.config.SortByContext && app.config.KeepCover
	if respectKeepCover {
		return fixTags || keepCover
	}
	return fixTags
}

func (app *application) saveTrack(track beatport.Track, directory string, quality string) (string, error) {
	var fileExtension string
	var displayQuality string

	var needledrop *beatport.TrackNeedledrop
	var stream *beatport.TrackStream

	switch app.config.Quality {
	case "medium-hls":
		trackNeedledrop, err := app.bp.StreamTrack(track.ID)
		if err != nil {
			return "", err
		}
		fileExtension = ".m4a"
		displayQuality = "AAC 128kbps - HLS"
		needledrop = trackNeedledrop
	default:
		trackStream, err := app.bp.DownloadTrack(track.ID, quality)
		if err != nil {
			return "", err
		}
		switch trackStream.StreamQuality {
		case ".128k.aac.mp4":
			fileExtension = ".m4a"
			displayQuality = "AAC 128kbps"
		case ".256k.aac.mp4":
			fileExtension = ".m4a"
			displayQuality = "AAC 256kbps"
		case ".flac":
			fileExtension = ".flac"
			displayQuality = "FLAC"
		default:
			return "", fmt.Errorf("invalid stream quality: %s", trackStream.StreamQuality)
		}
		stream = trackStream
	}
	fmt.Printf("Downloading %s (%s) [%s]\n", track.Name, track.MixName, displayQuality)

	fileName := track.Filename(
		app.config.TrackFileTemplate,
		app.config.WhitespaceCharacter,
		app.config.ArtistsLimit,
		app.config.ArtistsShortForm,
	)
	filePath := fmt.Sprintf("%s/%s%s", directory, fileName, fileExtension)

	if stream != nil {
		if err := app.downloadFile(stream.Location, filePath); err != nil {
			return "", err
		}
	} else if needledrop != nil {
		segments, key, err := beatport.GetStreamSegments(needledrop.Stream)
		if err != nil {
			return "", fmt.Errorf("get stream segments: %v", err)
		}
		segmentsFile, err := beatport.DownloadSegments(directory, *segments, *key)
		defer os.Remove(segmentsFile)
		if err != nil {
			return "", fmt.Errorf("download segments: %v", err)
		}
		if err := beatport.RemuxToM4A(segmentsFile, filePath); err != nil {
			return "", fmt.Errorf("remux to m4a: %v", err)
		}
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
		"IENG",
	}
)

func (app *application) tagTrack(location string, track beatport.Track, coverPath string) error {
	fileExt := filepath.Ext(location)
	if !app.config.FixTags {
		return nil
	}
	file, err := taglib.Read(location)
	if err != nil {
		return err
	}
	defer file.Close()

	if fileExt == ".flac" {
		date := file.GetProperty("RECORDING_DATE")
		key := file.GetProperty("INITIAL_KEY")
		for _, tag := range beatportTags {
			file.SetProperty(tag, "")
		}
		file.SetProperty("DATE", date)
		file.SetProperty("KEY", key)
		file.SetProperty("ALBUMARTIST", track.Release.ArtistsDisplay(
			beatport.ArtistTypeMain,
			app.config.ArtistsLimit,
			app.config.ArtistsShortForm,
		))
		file.SetProperty("CATALOGNUMBER", track.Release.CatalogNumber)
	}

	if fileExt == ".m4a" {
		file.SetProperty("TITLE", fmt.Sprintf("%s (%s)", track.Name, track.MixName))
		file.SetProperty("TRACKNUMBER", strconv.Itoa(track.Number))
		file.SetProperty("ALBUM", track.Release.Name)
		file.SetProperty("ARTIST", track.ArtistsDisplay(
			beatport.ArtistTypeMain,
			app.config.ArtistsLimit,
			app.config.ArtistsShortForm,
		))
		file.SetProperty("ALBUMARTIST", track.Release.ArtistsDisplay(
			beatport.ArtistTypeMain,
			app.config.ArtistsLimit,
			app.config.ArtistsShortForm,
		))
		file.SetProperty("CATALOGNUMBER", track.Release.CatalogNumber)
		file.SetProperty("DATE", track.PublishDate)
		file.SetProperty("BPM", strconv.Itoa(track.BPM))
		file.SetProperty("KEY", track.Key.Name)
		file.SetProperty("ISRC", track.ISRC)
		file.SetProperty("LABEL", track.Release.Label.Name)
	}

	if coverPath != "" && (app.config.CoverSize != config.DefaultCoverSize || fileExt == ".m4a") {
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
