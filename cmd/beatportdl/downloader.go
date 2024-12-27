package main

import (
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

func (app *application) errorLogWrapper(url, step string, err error) {
	app.LogError(fmt.Sprintf("[%s] %s", url, step), err)
}

func (app *application) infoLogWrapper(url, message string) {
	app.LogInfo(fmt.Sprintf("[%s] %s", url, message))
}

func (app *application) createDirectory(baseDir string, subDir ...string) (string, error) {
	fullPath := filepath.Join(baseDir, filepath.Join(subDir...))
	err := CreateDirectory(fullPath)
	return fullPath, err
}

type DownloadsDirectoryEntity interface {
	DirectoryName(template string, whitespace string, aLimit int, aShortForm string) string
}

func (app *application) setupDownloadsDirectory(baseDir string, entity DownloadsDirectoryEntity) (string, error) {
	if app.config.SortByContext {
		var subDir string
		switch castedEntity := entity.(type) {
		case *beatport.Release:
			subDir = castedEntity.DirectoryName(
				app.config.ReleaseDirectoryTemplate,
				app.config.WhitespaceCharacter,
				app.config.ArtistsLimit,
				app.config.ArtistsShortForm,
			)
			if app.config.SortByLabel && entity != nil {
				baseDir = filepath.Join(baseDir, castedEntity.Label.Name)
			}
		case *beatport.Playlist:
			subDir = castedEntity.DirectoryName(
				app.config.PlaylistDirectoryTemplate,
				app.config.WhitespaceCharacter,
				0,
				"",
			)
		case *beatport.Chart:
			subDir = castedEntity.DirectoryName(
				app.config.ChartDirectoryTemplate,
				app.config.WhitespaceCharacter,
				0,
				"",
			)
		case *beatport.Label:
			subDir = castedEntity.DirectoryName(
				app.config.LabelDirectoryTemplate,
				app.config.WhitespaceCharacter,
				0,
				"",
			)
		case *beatport.Artist:
			subDir = castedEntity.DirectoryName(
				app.config.ArtistDirectoryTemplate,
				app.config.WhitespaceCharacter,
				0,
				"",
			)
		}
		baseDir = filepath.Join(baseDir, subDir)
	}
	return app.createDirectory(baseDir)
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
	if err != nil {
		os.Remove(coverPath)
		return "", err
	}
	return coverPath, nil
}

func (app *application) handleCoverFile(path string) error {
	if path == "" {
		return nil
	}
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

var (
	ErrTrackFileExists = errors.New("file already exists")
)

func (app *application) saveTrack(track beatport.Track, directory string, quality string) (string, error) {
	var fileExtension string
	var displayQuality string

	var stream *beatport.TrackStream
	var download *beatport.TrackDownload

	switch app.config.Quality {
	case "medium-hls":
		trackStream, err := app.bp.StreamTrack(track.ID)
		if err != nil {
			return "", err
		}
		fileExtension = ".m4a"
		displayQuality = "AAC 128kbps - HLS"
		stream = trackStream
	default:
		trackDownload, err := app.bp.DownloadTrack(track.ID, quality)
		if err != nil {
			return "", err
		}
		switch trackDownload.StreamQuality {
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
			return "", fmt.Errorf("invalid stream quality: %s", trackDownload.StreamQuality)
		}
		download = trackDownload
	}

	fileName := track.Filename(
		app.config.TrackFileTemplate,
		app.config.WhitespaceCharacter,
		app.config.ArtistsLimit,
		app.config.ArtistsShortForm,
		app.config.KeySystem,
	)
	filePath := fmt.Sprintf("%s/%s%s", directory, fileName, fileExtension)
	if _, err := os.Stat(filePath); err == nil {
		switch app.config.TrackExists {
		case "skip":
			return "", nil
		case "update":
			app.infoLogWrapper(track.StoreUrl(), "updating tags")
			return filePath, nil
		case "error":
			return "", ErrTrackFileExists
		}
	}

	var prefix string
	infoDisplay := fmt.Sprintf("%s (%s) [%s]", track.Name.String(), track.MixName.String(), displayQuality)
	if app.config.ShowProgress {
		prefix = infoDisplay
	} else {
		fmt.Println("Downloading " + infoDisplay)
	}

	if download != nil {
		if err := app.downloadFile(download.Location, filePath, prefix); err != nil {
			return "", err
		}
	} else if stream != nil {
		segments, key, err := getStreamSegments(stream.Url)
		if err != nil {
			return "", fmt.Errorf("get stream segments: %v", err)
		}
		segmentsFile, err := app.downloadSegments(directory, *segments, *key, prefix)
		defer os.Remove(segmentsFile)
		if err != nil {
			return "", fmt.Errorf("download segments: %v", err)
		}
		if err := remuxToM4A(segmentsFile, filePath); err != nil {
			return "", fmt.Errorf("remux to m4a: %v", err)
		}
	}

	if !app.config.ShowProgress {
		fmt.Printf("Finished downloading %s\n", infoDisplay)
	}

	return filePath, nil
}

const (
	rawTagSuffix = "_raw"
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

	subgenre := ""
	genreWithSubgenre := track.Genre.Name
	subgenreOrGenre := track.Genre.Name
	if track.Subgenre != nil {
		subgenre = track.Subgenre.Name
		genreWithSubgenre = genreWithSubgenre + " | " + subgenre
		subgenreOrGenre = subgenre
	}
	mappingValues := map[string]string{
		"track_id":   strconv.Itoa(int(track.ID)),
		"track_url":  track.StoreUrl(),
		"track_name": fmt.Sprintf("%s (%s)", track.Name.String(), track.MixName.String()),
		"track_artists": track.Artists.Display(
			0,
			"",
		),
		"track_number":              strconv.Itoa(track.Number),
		"track_number_with_total":   fmt.Sprintf("%d/%d", track.Number, track.Release.TrackCount),
		"track_genre":               track.Genre.Name,
		"track_subgenre":            subgenre,
		"track_genre_with_subgenre": genreWithSubgenre,
		"track_subgenre_or_genre":   subgenreOrGenre,
		"track_key":                 track.Key.Display(app.config.KeySystem),
		"track_bpm":                 strconv.Itoa(track.BPM),
		"track_isrc":                track.ISRC,

		"release_id":   strconv.Itoa(int(track.Release.ID)),
		"release_url":  track.Release.StoreUrl(),
		"release_name": track.Release.Name.String(),
		"release_artists": track.Release.Artists.Display(
			0,
			"",
		),
		"release_date":           track.Release.Date,
		"release_year":           track.Release.Year(),
		"release_track_count":    strconv.Itoa(track.Release.TrackCount),
		"release_catalog_number": track.Release.CatalogNumber.String(),
		"release_upc":            track.Release.UPC,
		"release_label":          track.Release.Label.Name,
		"release_label_url":      track.Release.Label.StoreUrl(),
	}

	existingTags, err := file.PropertyKeys()
	if err != nil {
		return fmt.Errorf("read existing tags: %v", err)
	}

	for _, tag := range existingTags {
		file.SetProperty(tag, nil)
	}

	if fileExt == ".flac" {
		for field, property := range app.config.TagMappings["flac"] {
			value := mappingValues[field]
			if value != "" {
				file.SetProperty(property, &value)
			}
		}
	} else if fileExt == ".m4a" {
		rawTags := make(map[string]string)

		for field, property := range app.config.TagMappings["m4a"] {
			if strings.HasSuffix(property, rawTagSuffix) {
				if mappingValues[field] != "" {
					property = strings.TrimSuffix(property, rawTagSuffix)
					rawTags[property] = mappingValues[field]
				}
			} else {
				value := mappingValues[field]
				if value != "" {
					file.SetProperty(property, &value)
				}
			}
		}

		for tag, value := range rawTags {
			file.SetItemMp4(tag, value)
		}
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
		return fmt.Errorf("save track: %v", err)
	}
	if err = app.tagTrack(location, *track, coverPath); err != nil && location != "" {
		return fmt.Errorf("tag track: %v", err)
	}
	return nil
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
		app.errorLogWrapper(url, "parse url", err)
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
		app.errorLogWrapper(url, "fetch track", err)
		return
	}

	release, err := app.bp.GetRelease(track.Release.ID)
	if err != nil {
		app.errorLogWrapper(url, "fetch track release", err)
		return
	}
	track.Release = *release

	downloadsDir, err := app.setupDownloadsDirectory(app.config.DownloadsDirectory, release)
	if err != nil {
		app.errorLogWrapper(url, "setup downloads directory", err)
		return
	}

	wg := sync.WaitGroup{}
	app.downloadWorker(&wg, func() {
		var cover string
		if app.requireCover(true, true) {
			cover, err = app.downloadCover(track.Release.Image, downloadsDir)
			if err != nil {
				app.errorLogWrapper(url, "download track release cover", err)
			}
		}

		if err := app.handleTrack(track, downloadsDir, cover); err != nil {
			app.errorLogWrapper(url, "handle track", err)
			return
		}

		if err := app.handleCoverFile(cover); err != nil {
			app.errorLogWrapper(url, "handle cover file", err)
			return
		}
	})
	wg.Wait()
}

func (app *application) handleReleaseLink(url string, link beatport.Link) {
	release, err := app.bp.GetRelease(link.ID)
	if err != nil {
		app.errorLogWrapper(url, "fetch release", err)
		return
	}

	downloadsDir, err := app.setupDownloadsDirectory(app.config.DownloadsDirectory, release)
	if err != nil {
		app.errorLogWrapper(url, "setup downloads directory", err)
		return
	}

	var cover string
	if app.requireCover(true, true) {
		app.semAcquire(app.downloadSem)
		cover, err = app.downloadCover(release.Image, downloadsDir)
		if err != nil {
			app.errorLogWrapper(url, "download release cover", err)
		}
		app.semRelease(app.downloadSem)
	}

	wg := sync.WaitGroup{}
	for _, trackUrl := range release.TrackUrls {
		app.downloadWorker(&wg, func() {
			trackLink, _ := app.bp.ParseUrl(trackUrl)
			track, err := app.bp.GetTrack(trackLink.ID)
			if err != nil {
				app.errorLogWrapper(trackUrl, "fetch release track", err)
				return
			}
			trackStoreUrl := track.StoreUrl()
			track.Release = *release

			if err := app.handleTrack(track, downloadsDir, cover); err != nil {
				app.errorLogWrapper(trackStoreUrl, "handle track", err)
				return
			}
		})
	}
	wg.Wait()

	if err := app.handleCoverFile(cover); err != nil {
		app.errorLogWrapper(url, "handle cover file", err)
		return
	}
}

func (app *application) handlePlaylistLink(url string, link beatport.Link) {
	playlist, err := app.bp.GetPlaylist(link.ID)
	if err != nil {
		app.errorLogWrapper(url, "fetch playlist", err)
		return
	}

	downloadsDir, err := app.setupDownloadsDirectory(app.config.DownloadsDirectory, playlist)
	if err != nil {
		app.errorLogWrapper(url, "setup downloads directory", err)
		return
	}

	wg := sync.WaitGroup{}
	err = ForPaginated[beatport.PlaylistItem](link.ID, app.bp.GetPlaylistItems, func(item beatport.PlaylistItem, i int) error {
		app.downloadWorker(&wg, func() {
			item.Track.Number = item.Position
			trackStoreUrl := item.Track.StoreUrl()

			release, err := app.bp.GetRelease(item.Track.Release.ID)
			if err != nil {
				app.errorLogWrapper(trackStoreUrl, "fetch track release", err)
				return
			}
			item.Track.Release = *release

			var cover string
			if app.requireCover(true, false) {
				cover, err = app.downloadCover(item.Track.Release.Image, downloadsDir)
				if err != nil {
					app.errorLogWrapper(trackStoreUrl, "download track release cover", err)
				} else {
					defer os.Remove(cover)
				}
			}

			if err := app.handleTrack(&item.Track, downloadsDir, cover); err != nil {
				app.errorLogWrapper(trackStoreUrl, "handle track", err)
				return
			}
		})
		return nil
	})

	if err != nil {
		app.errorLogWrapper(url, "handle playlist items", err)
		return
	}

	wg.Wait()
}

func (app *application) handleChartLink(url string, link beatport.Link) {
	chart, err := app.bp.GetChart(link.ID)
	if err != nil {
		app.errorLogWrapper(url, "fetch chart", err)
		return
	}

	downloadsDir, err := app.setupDownloadsDirectory(app.config.DownloadsDirectory, chart)
	if err != nil {
		app.errorLogWrapper(url, "setup downloads directory", err)
		return
	}
	wg := sync.WaitGroup{}

	if app.requireCover(false, true) {
		app.downloadWorker(&wg, func() {
			cover, err := app.downloadCover(chart.Image, downloadsDir)
			if err != nil {
				app.errorLogWrapper(url, "download chart cover", err)
			}
			if err := app.handleCoverFile(cover); err != nil {
				app.errorLogWrapper(url, "handle cover file", err)
				return
			}
		})
	}

	err = ForPaginated[beatport.Track](link.ID, app.bp.GetChartTracks, func(track beatport.Track, i int) error {
		app.downloadWorker(&wg, func() {
			track.Number = i + 1
			trackStoreUrl := track.StoreUrl()

			release, err := app.bp.GetRelease(track.Release.ID)
			if err != nil {
				app.errorLogWrapper(trackStoreUrl, "fetch track release", err)
				return
			}
			track.Release = *release

			var cover string
			if app.requireCover(true, false) {
				cover, err = app.downloadCover(track.Release.Image, downloadsDir)
				if err != nil {
					app.errorLogWrapper(trackStoreUrl, "download track release cover", err)
				} else {
					defer os.Remove(cover)
				}
			}

			if err := app.handleTrack(&track, downloadsDir, cover); err != nil {
				app.errorLogWrapper(trackStoreUrl, "handle track", err)
				return
			}
		})
		return nil
	})

	if err != nil {
		app.errorLogWrapper(url, "handle playlist items", err)
		return
	}

	wg.Wait()
}

func (app *application) handleLabelLink(url string, link beatport.Link) {
	label, err := app.bp.GetLabel(link.ID)
	if err != nil {
		app.errorLogWrapper(url, "fetch label", err)
		return
	}

	downloadsDir, err := app.setupDownloadsDirectory(app.config.DownloadsDirectory, label)
	if err != nil {
		app.errorLogWrapper(url, "setup downloads directory", err)
		return
	}

	err = ForPaginated[beatport.Release](link.ID, app.bp.GetLabelReleases, func(release beatport.Release, i int) error {
		app.background(func() {
			releaseStoreUrl := release.StoreUrl()
			releaseDir, err := app.setupDownloadsDirectory(downloadsDir, &release)
			if err != nil {
				app.errorLogWrapper(releaseStoreUrl, "setup release downloads directory", err)
				return
			}

			var cover string
			if app.requireCover(true, true) {
				app.semAcquire(app.downloadSem)
				cover, err = app.downloadCover(release.Image, releaseDir)
				if err != nil {
					app.errorLogWrapper(releaseStoreUrl, "download release cover", err)
				}
				app.semRelease(app.downloadSem)
			}

			wg := sync.WaitGroup{}
			err = ForPaginated[beatport.Track](release.ID, app.bp.GetReleaseTracks, func(track beatport.Track, i int) error {
				app.downloadWorker(&wg, func() {
					trackStoreUrl := track.StoreUrl()
					t, err := app.bp.GetTrack(track.ID)
					if err != nil {
						app.errorLogWrapper(trackStoreUrl, "fetch full track", err)
						return
					}
					t.Release = release

					if err := app.handleTrack(t, releaseDir, cover); err != nil {
						app.errorLogWrapper(trackStoreUrl, "handle track", err)
						return
					}
				})
				return nil
			})
			if err != nil {
				app.errorLogWrapper(releaseStoreUrl, "handle release tracks", err)
				return
			}
			wg.Wait()

			if err := app.handleCoverFile(cover); err != nil {
				app.errorLogWrapper(releaseStoreUrl, "handle cover file", err)
				return
			}
		})
		return nil
	})

	if err != nil {
		app.errorLogWrapper(url, "handle label releases", err)
		return
	}
}

func (app *application) handleArtistLink(url string, link beatport.Link) {
	artist, err := app.bp.GetArtist(link.ID)
	if err != nil {
		app.errorLogWrapper(url, "fetch artist", err)
		return
	}

	downloadsDir, err := app.setupDownloadsDirectory(app.config.DownloadsDirectory, artist)
	if err != nil {
		app.errorLogWrapper(url, "setup downloads directory", err)
		return
	}

	wg := sync.WaitGroup{}
	err = ForPaginated[beatport.Track](link.ID, app.bp.GetArtistTracks, func(track beatport.Track, i int) error {
		app.downloadWorker(&wg, func() {
			trackStoreUrl := track.StoreUrl()
			t, err := app.bp.GetTrack(track.ID)
			if err != nil {
				app.errorLogWrapper(trackStoreUrl, "fetch full track", err)
				return
			}

			release, err := app.bp.GetRelease(track.Release.ID)
			if err != nil {
				app.errorLogWrapper(trackStoreUrl, "fetch track release", err)
				return
			}
			t.Release = *release

			releaseDir, err := app.setupDownloadsDirectory(downloadsDir, release)
			if err != nil {
				app.errorLogWrapper(trackStoreUrl, "setup track release downloads directory", err)
				return
			}

			var cover string
			if app.requireCover(true, true) {
				cover, err = app.downloadCover(release.Image, releaseDir)
				if err != nil {
					app.errorLogWrapper(trackStoreUrl, "download track release cover", err)
				}
			}

			if err := app.handleTrack(t, releaseDir, cover); err != nil {
				app.errorLogWrapper(trackStoreUrl, "handle track", err)
				return
			}

			if err := app.handleCoverFile(cover); err != nil {
				app.errorLogWrapper(trackStoreUrl, "handle cover file", err)
				return
			}
		})
		return nil
	})
	if err != nil {
		app.errorLogWrapper(url, "handle artist tracks", err)
		return
	}

	wg.Wait()
}
