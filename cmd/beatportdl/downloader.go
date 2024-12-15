package main

import (
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
	"unspok3n/beatportdl/internal/taglib"
)

func (app *application) logWrapper(url, step string, err error) {
	app.LogError(fmt.Sprintf("[%s] %s", url, step), err)
}

func (app *application) createDirectory(baseDir string, subDir ...string) (string, error) {
	fullPath := filepath.Join(baseDir, filepath.Join(subDir...))
	err := CreateDirectory(fullPath)
	return fullPath, err
}

func (app *application) setupBasicDownloadsDirectory(baseDir string, release *beatport.Release) (string, error) {
	dir := baseDir
	if app.config.SortByContext {
		subDir := release.DirectoryName(
			app.config.ReleaseDirectoryTemplate,
			app.config.WhitespaceCharacter,
			app.config.ArtistsLimit,
			app.config.ArtistsShortForm,
		)
		if app.config.SortByLabel && release != nil {
			dir = filepath.Join(dir, release.Label.NameSanitized())
		}
		dir = filepath.Join(dir, subDir)
	}
	return app.createDirectory(dir)
}

func (app *application) setupCustomDownloadsDirectory(baseDir string, subDir string) (string, error) {
	dir := baseDir
	if app.config.SortByContext {
		dir = filepath.Join(dir, subDir)
	}
	return app.createDirectory(dir)
}

func (app *application) requireCover(respectFixTags, respectKeepCover bool) bool {
	fixTags := respectFixTags && app.config.FixTags &&
		(app.config.CoverSize != config.DefaultCoverSize || app.config.Quality != "lossless")
	keepCover := respectKeepCover && app.config.SortByContext && app.config.KeepCover
	return fixTags || keepCover
}

func (app *application) downloadCover(image beatport.Image, downloadsDir string) (string, error) {
	coverUrl := image.FormattedUrl(app.config.CoverSize)
	coverPath := filepath.Join(downloadsDir, uuid.New().String())
	err := app.downloadFile(coverUrl, coverPath, "")
	return coverPath, err
}

func (app *application) handleCoverFile(path string) error {
	if app.config.KeepCover && app.config.SortByContext {
		newPath := filepath.Dir(path) + "/cover.jpg"
		if err := os.Rename(path, newPath); err != nil {
			return err
		}
	} else {
		os.Remove(path)
	}
	return nil
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

	var prefix string
	infoDisplay := fmt.Sprintf("%s (%s) [%s]", track.Name.String(), track.MixName.String(), displayQuality)
	if app.config.ShowProgress {
		prefix = infoDisplay
	} else {
		fmt.Println("Downloading " + infoDisplay)
	}

	fileName := track.Filename(
		app.config.TrackFileTemplate,
		app.config.WhitespaceCharacter,
		app.config.ArtistsLimit,
		app.config.ArtistsShortForm,
		app.config.KeySystem,
	)
	filePath := fmt.Sprintf("%s/%s%s", directory, fileName, fileExtension)

	if stream != nil {
		if err := app.downloadFile(stream.Location, filePath, prefix); err != nil {
			return "", err
		}
	} else if needledrop != nil {
		segments, key, err := GetStreamSegments(needledrop.Stream)
		if err != nil {
			return "", fmt.Errorf("get stream segments: %v", err)
		}
		segmentsFile, err := app.downloadSegments(directory, *segments, *key, prefix)
		defer os.Remove(segmentsFile)
		if err != nil {
			return "", fmt.Errorf("download segments: %v", err)
		}
		if err := RemuxToM4A(segmentsFile, filePath); err != nil {
			return "", fmt.Errorf("remux to m4a: %v", err)
		}
	}

	if !app.config.ShowProgress {
		fmt.Printf("Finished downloading %s\n", infoDisplay)
	}

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
		for _, tag := range beatportTags {
			file.SetProperty(tag, "")
		}
		file.SetProperty("TITLE", fmt.Sprintf("%s (%s)", track.Name.String(), track.MixName.String()))
		file.SetProperty("TRACKNUMBER", strconv.Itoa(track.Number))
		file.SetProperty("ALBUM", track.Release.Name.String())
		file.SetProperty("CATALOGNUMBER", track.Release.CatalogNumber.String())
		file.SetProperty("DATE", track.Release.Date)
		file.SetProperty("KEY", track.Key.Display(app.config.KeySystem))
		file.SetProperty("ALBUMARTIST", track.Release.Artists.Display(
			0,
			"",
		))
		file.SetProperty("CATALOGNUMBER", track.Release.CatalogNumber.String())
	}

	if fileExt == ".m4a" {
		file.SetProperty("TITLE", fmt.Sprintf("%s (%s)", track.Name.String(), track.MixName.String()))
		file.SetProperty("TRACKNUMBER", strconv.Itoa(track.Number))
		file.SetProperty("ALBUM", track.Release.Name.String())
		file.SetProperty("ARTIST", track.Artists.Display(
			0,
			"",
		))
		file.SetProperty("ALBUMARTIST", track.Release.Artists.Display(
			0,
			"",
		))
		file.SetProperty("CATALOGNUMBER", track.Release.CatalogNumber.String())
		file.SetProperty("DATE", track.PublishDate)
		file.SetProperty("BPM", strconv.Itoa(track.BPM))
		file.SetProperty("KEY", track.Key.Display(app.config.KeySystem))
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

func (app *application) handleTrack(track *beatport.Track, downloadsDir string, coverPath string) error {
	location, err := app.saveTrack(*track, downloadsDir, app.config.Quality)
	if err != nil {
		return err
	}
	return app.tagTrack(location, *track, coverPath)
}

func ForPaginated[T any](
	entityId int64,
	fetchPage func(id int64, page int) (results *beatport.Paginated[T], err error),
	processItem func(item T, i int) error,
) error {
	page := 1
	for {
		paginated, err := fetchPage(entityId, page)
		if err != nil {
			return fmt.Errorf("fetch page: %w", err)
		}

		for i, item := range paginated.Results {
			if err := processItem(item, i); err != nil {
				return fmt.Errorf("process item: %w", err)
			}
		}

		if paginated.Next == nil {
			break
		}
		page++
	}
	return nil
}

func (app *application) handleUrl(url string) {
	link, err := app.bp.ParseUrl(url)
	if err != nil {
		app.logWrapper(url, "parse url", err)
		return
	}

	switch link.Type {
	case beatport.TrackLink:
		app.handleTrackLink(url, *link)
	case beatport.ReleaseLink:
		app.handleReleaseLink(url, *link)
	case beatport.PlaylistLink:
		app.handlePlaylistLink(url, *link)
	case beatport.ChartLink:
		app.handleChartLink(url, *link)
	case beatport.LabelLink:
		app.handleLabelLink(url, *link)
	case beatport.ArtistLink:
		app.handleArtistLink(url, *link)
	default:
		app.LogError("handle URL", ErrUnsupportedLinkType)
	}
}

func (app *application) handleTrackLink(url string, link beatport.Link) {
	track, err := app.bp.GetTrack(link.ID)
	if err != nil {
		app.logWrapper(url, "fetch track", err)
		return
	}

	release, err := app.bp.GetRelease(track.Release.ID)
	if err != nil {
		app.logWrapper(url, "fetch track release", err)
		return
	}
	track.Release = *release

	downloadsDir, err := app.setupBasicDownloadsDirectory(app.config.DownloadsDirectory, release)
	if err != nil {
		app.logWrapper(url, "setup downloads directory", err)
		return
	}

	wg := sync.WaitGroup{}
	app.downloadWorker(&wg, func() {
		var cover string
		if app.requireCover(true, true) {
			cover, err = app.downloadCover(track.Release.Image, downloadsDir)
			if err != nil {
				app.logWrapper(url, "download track release cover", err)
				return
			}
		}

		if err := app.handleTrack(track, downloadsDir, cover); err != nil {
			app.logWrapper(url, "handle track", err)
			return
		}

		if err := app.handleCoverFile(cover); err != nil {
			app.logWrapper(url, "handle cover file", err)
			return
		}
	})
	wg.Wait()
}

func (app *application) handleReleaseLink(url string, link beatport.Link) {
	release, err := app.bp.GetRelease(link.ID)
	if err != nil {
		app.logWrapper(url, "fetch release", err)
		return
	}

	downloadsDir, err := app.setupBasicDownloadsDirectory(app.config.DownloadsDirectory, release)
	if err != nil {
		app.logWrapper(url, "setup downloads directory", err)
		return
	}

	var cover string
	if app.requireCover(true, true) {
		app.semAcquire(app.downloadSem)
		cover, err = app.downloadCover(release.Image, downloadsDir)
		if err != nil {
			app.semRelease(app.downloadSem)
			app.logWrapper(url, "download track release cover", err)
			return
		}
		app.semRelease(app.downloadSem)
	}

	wg := sync.WaitGroup{}
	for _, trackUrl := range release.TrackUrls {
		app.downloadWorker(&wg, func() {
			trackLink, _ := app.bp.ParseUrl(trackUrl)
			track, err := app.bp.GetTrack(trackLink.ID)
			if err != nil {
				step := fmt.Sprintf("fetch release track '%d'", trackLink.ID)
				app.logWrapper(url, step, err)
				return
			}
			track.Release = *release

			if err := app.handleTrack(track, downloadsDir, cover); err != nil {
				app.logWrapper(url, "handle track", err)
				return
			}
		})
	}
	wg.Wait()

	if err := app.handleCoverFile(cover); err != nil {
		app.logWrapper(url, "handle cover file", err)
		return
	}
}

func (app *application) handlePlaylistLink(url string, link beatport.Link) {
	playlist, err := app.bp.GetPlaylist(link.ID)
	if err != nil {
		app.logWrapper(url, "fetch playlist", err)
		return
	}

	downloadsDir, err := app.setupCustomDownloadsDirectory(
		app.config.DownloadsDirectory,
		playlist.Name,
	)
	if err != nil {
		app.logWrapper(url, "setup downloads directory", err)
		return
	}

	wg := sync.WaitGroup{}
	err = ForPaginated[beatport.PlaylistItem](link.ID, app.bp.GetPlaylistItems, func(item beatport.PlaylistItem, i int) error {
		app.downloadWorker(&wg, func() {
			item.Track.Number = item.Position

			release, err := app.bp.GetRelease(item.Track.Release.ID)
			if err != nil {
				step := fmt.Sprintf("fetch track release '%d'", item.Track.ID)
				app.logWrapper(url, step, err)
				return
			}
			item.Track.Release = *release

			var cover string
			if app.requireCover(true, false) {
				cover, err = app.downloadCover(item.Track.Release.Image, downloadsDir)
				if err != nil {
					app.logWrapper(url, "download track release cover", err)
					return
				}
				defer os.Remove(cover)
			}

			if err := app.handleTrack(&item.Track, downloadsDir, cover); err != nil {
				app.logWrapper(url, "handle track", err)
				return
			}
		})
		return nil
	})

	if err != nil {
		app.logWrapper(url, "handle playlist items", err)
		return
	}

	wg.Wait()
}

func (app *application) handleChartLink(url string, link beatport.Link) {
	chart, err := app.bp.GetChart(link.ID)
	if err != nil {
		app.logWrapper(url, "fetch chart", err)
		return
	}

	downloadsDir, err := app.setupCustomDownloadsDirectory(
		app.config.DownloadsDirectory,
		chart.Name,
	)
	if err != nil {
		app.logWrapper(url, "setup downloads directory", err)
		return
	}
	wg := sync.WaitGroup{}

	if app.requireCover(false, true) {
		app.downloadWorker(&wg, func() {
			cover, err := app.downloadCover(chart.Image, downloadsDir)
			if err != nil {
				app.logWrapper(url, "download chart cover", err)
				return
			}
			if err := app.handleCoverFile(cover); err != nil {
				app.logWrapper(url, "handle cover file", err)
				return
			}
		})
	}

	err = ForPaginated[beatport.Track](link.ID, app.bp.GetChartTracks, func(track beatport.Track, i int) error {
		app.downloadWorker(&wg, func() {
			track.Number = i + 1

			release, err := app.bp.GetRelease(track.Release.ID)
			if err != nil {
				step := fmt.Sprintf("fetch track release '%d'", track.ID)
				app.logWrapper(url, step, err)
				return
			}
			track.Release = *release

			var cover string
			if app.requireCover(true, false) {
				cover, err = app.downloadCover(track.Release.Image, downloadsDir)
				if err != nil {
					app.logWrapper(url, "download track release cover", err)
					return
				}
				defer os.Remove(cover)
			}

			if err := app.handleTrack(&track, downloadsDir, cover); err != nil {
				app.logWrapper(url, "handle track", err)
				return
			}
		})
		return nil
	})

	if err != nil {
		app.logWrapper(url, "handle playlist items", err)
		return
	}

	wg.Wait()
}

func (app *application) handleLabelLink(url string, link beatport.Link) {
	label, err := app.bp.GetLabel(link.ID)
	if err != nil {
		app.logWrapper(url, "fetch label", err)
		return
	}

	downloadsDir, err := app.setupCustomDownloadsDirectory(
		app.config.DownloadsDirectory,
		label.NameSanitized(),
	)
	if err != nil {
		app.logWrapper(url, "setup downloads directory", err)
		return
	}

	err = ForPaginated[beatport.Release](link.ID, app.bp.GetLabelReleases, func(release beatport.Release, i int) error {
		app.background(func() {
			releaseDir, err := app.setupBasicDownloadsDirectory(
				downloadsDir,
				&release,
			)
			if err != nil {
				app.logWrapper(url, "setup release download directory", err)
				return
			}

			var cover string
			if app.requireCover(true, true) {
				app.semAcquire(app.downloadSem)
				cover, err = app.downloadCover(release.Image, releaseDir)
				if err != nil {
					app.semRelease(app.downloadSem)
					app.logWrapper(url, "download release cover", err)
					return
				}
				app.semRelease(app.downloadSem)
			}

			wg := sync.WaitGroup{}
			err = ForPaginated[beatport.Track](release.ID, app.bp.GetReleaseTracks, func(track beatport.Track, i int) error {
				app.downloadWorker(&wg, func() {
					t, err := app.bp.GetTrack(track.ID)
					if err != nil {
						step := fmt.Sprintf("fetch full track '%d'", track.ID)
						app.logWrapper(url, step, err)
						return
					}
					t.Release = release

					if err := app.handleTrack(t, releaseDir, cover); err != nil {
						app.logWrapper(url, "handle track", err)
						return
					}
				})
				return nil
			})
			if err != nil {
				app.logWrapper(url, "handle release tracks", err)
				return
			}
			wg.Wait()

			if err := app.handleCoverFile(cover); err != nil {
				app.logWrapper(url, "handle cover file", err)
				return
			}
		})
		return nil
	})

	if err != nil {
		app.logWrapper(url, "handle label releases", err)
		return
	}
}

func (app *application) handleArtistLink(url string, link beatport.Link) {
	artist, err := app.bp.GetArtist(link.ID)
	if err != nil {
		app.logWrapper(url, "fetch artist", err)
		return
	}

	downloadsDir, err := app.setupCustomDownloadsDirectory(
		app.config.DownloadsDirectory,
		artist.NameSanitized(),
	)
	if err != nil {
		app.logWrapper(url, "setup downloads directory", err)
		return
	}

	wg := sync.WaitGroup{}
	err = ForPaginated[beatport.Track](link.ID, app.bp.GetArtistTracks, func(track beatport.Track, i int) error {
		app.downloadWorker(&wg, func() {
			t, err := app.bp.GetTrack(track.ID)
			if err != nil {
				step := fmt.Sprintf("fetch full track '%d'", track.ID)
				app.logWrapper(url, step, err)
				return
			}

			release, err := app.bp.GetRelease(track.Release.ID)
			if err != nil {
				step := fmt.Sprintf("fetch track release '%d'", track.ID)
				app.logWrapper(url, step, err)
				return
			}
			t.Release = *release

			releaseDir, err := app.setupBasicDownloadsDirectory(
				downloadsDir,
				release,
			)
			if err != nil {
				app.logWrapper(url, "setup release download directory", err)
				return
			}

			var cover string
			if app.requireCover(true, true) {
				cover, err = app.downloadCover(release.Image, releaseDir)
				if err != nil {
					app.logWrapper(url, "download track release cover", err)
					return
				}
			}

			if err := app.handleTrack(t, releaseDir, cover); err != nil {
				app.logWrapper(url, "handle track", err)
				return
			}

			if err := app.handleCoverFile(cover); err != nil {
				app.logWrapper(url, "handle cover file", err)
				return
			}
		})
		return nil
	})
	if err != nil {
		app.logWrapper(url, "handle artist tracks", err)
		return
	}

	wg.Wait()
}
