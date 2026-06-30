package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unspok3n/beatportdl/config"
	"unspok3n/beatportdl/internal/beatport"
)

var (
	ErrUnsupportedLinkType = errors.New("unsupported link type")
)

func Setup() (cfg *config.AppConfig, cachePath string, err error) {
	configFilePath, exists, err := FindConfigFile()
	if err != nil {
		return nil, "", err
	}

	if !exists {
		fmt.Println("Config file not found, creating a new one:", configFilePath)

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

	cacheFilePath, exists, err := FindCacheFile()
	if err != nil {
		return nil, configFilePath, fmt.Errorf("get executable path: %w", err)
	}

	return parsedConfig, cacheFilePath, nil
}

func (app *application) mainPrompt() {
	fmt.Print("Enter url or search query: ")
	input := GetLine()
	if strings.HasPrefix(input, "https://www.beatport.com") {
		if strings.Contains(input, "/label/") || strings.Contains(input, "/artist/") {
			app.filtersPrompt(input)
		} else {
			app.urls = append(app.urls, input)
		}
	} else {
		app.search(input)
	}
}

func (app *application) filtersPrompt(rawURL string) {
	link, err := app.bp.ParseUrl(rawURL)
	if err != nil {
		fmt.Println("Could not parse URL:", err)
		return
	}

	params := "include_facets=true&per_page=1"

	var stats *entityStats
	var listItemName string
	switch link.Type {
	case beatport.LabelLink:
		listItemName = "releases"
		labelReleases, err := app.bp.GetLabelReleases(link.ID, 1, params)
		if err != nil {
			fmt.Println("Could not fetch label releases:", err)
			return
		}

		stats = newEntityStats(labelReleases.Count, &labelReleases.Facets)
	case beatport.ArtistLink:
		listItemName = "tracks"
		artistTracks, err := app.bp.GetArtistTracks(link.ID, 1, params)
		if err != nil {
			fmt.Println("Could not fetch artist tracks:", err)
			return
		}

		stats = newEntityStats(artistTracks.Count, &artistTracks.Facets)
	default:
		return
	}

	fmt.Printf("\n%d %s total.\n", stats.total, listItemName)

	genres := rankMap(stats.genres)
	subgenres := rankMap(stats.subgenres)
	artists := rankMap(stats.artists)

	const (
		stepGenres = iota
		stepSubgenres
		stepArtists
		stepDateFrom
		stepDateTo
		stepConfirm
	)

	var selectedGenres, selectedSubgenres, selectedArtists []string
	var dateFrom, dateTo string

	step := stepGenres
	for {
		switch step {

		case stepGenres:
			sel, back := selectFromList("\nGenres", genres)
			if back {
				fmt.Println("Cancelled.")
				return
			}
			selectedGenres = sel
			step = stepSubgenres

		case stepSubgenres:
			if len(subgenres) == 0 {
				step = stepArtists
				continue
			}
			sel, back := selectFromList("\nSubgenres", subgenres)
			if back {
				step = stepGenres
				continue
			}
			selectedSubgenres = sel
			step = stepArtists

		case stepArtists:
			if len(artists) == 0 {
				step = stepDateFrom
				continue
			}
			sel, back := selectFromList("\nArtists (by track count)", artists)
			if back {
				if len(subgenres) > 0 {
					step = stepSubgenres
				} else {
					step = stepGenres
				}
				continue
			}
			selectedArtists = sel
			step = stepDateFrom

		case stepDateFrom:
			fmt.Print("\nDownload from date (e.g. 1996 or 1996-06-01, Enter for all, b to go back): ")
			input := strings.TrimSpace(GetLine())
			if input == "b" {
				if len(artists) > 0 {
					step = stepArtists
				} else if len(subgenres) > 0 {
					step = stepSubgenres
				} else {
					step = stepGenres
				}
				continue
			}
			dateFrom = normaliseDate(input)
			step = stepDateTo

		case stepDateTo:
			fmt.Print("Download up to date   (e.g. 2024 or 2024-12-31, Enter for all, b to go back): ")
			input := strings.TrimSpace(GetLine())
			if input == "b" {
				step = stepDateFrom
				continue
			}
			dateTo = normaliseDateTo(input)
			step = stepConfirm

		case stepConfirm:
			fmt.Println("\n--- Download filter summary ---")
			if len(selectedGenres) > 0 {
				fmt.Println("  Genres:    ", strings.Join(selectedGenres, ", "))
			} else {
				fmt.Println("  Genres:     all")
			}
			if len(selectedSubgenres) > 0 {
				fmt.Println("  Subgenres: ", strings.Join(selectedSubgenres, ", "))
			} else {
				fmt.Println("  Subgenres:  all")
			}
			if len(selectedArtists) > 0 {
				fmt.Println("  Artists:   ", strings.Join(selectedArtists, ", "))
			} else {
				fmt.Println("  Artists:    all")
			}
			dateRange := "all time"
			if dateFrom != "" && dateTo != "" {
				dateRange = dateFrom + " → " + dateTo
			} else if dateFrom != "" {
				dateRange = dateFrom + " → present"
			} else if dateTo != "" {
				dateRange = "up to " + dateTo
			}
			fmt.Println("  Dates:     ", dateRange)

			fmt.Print("\nStart download? (y/n/b to go back): ")
			ans := strings.ToLower(strings.TrimSpace(GetLine()))
			if ans == "b" {
				step = stepDateTo
				continue
			}
			if ans != "y" {
				fmt.Println("Cancelled.")
				return
			}

			var params []string

			for i, genre := range selectedGenres {
				selectedGenres[i] = url.QueryEscape(genre)
			}
			genresStr := strings.Join(selectedGenres, ",")

			var subgenreIds []string
			for _, subgenre := range selectedSubgenres {
				subgenreIds = append(subgenreIds, strconv.FormatInt(stats.subgenreIds[subgenre], 10))
			}
			subgenreIdsStr := strings.Join(subgenreIds, ",")

			for i, artist := range selectedArtists {
				selectedArtists[i] = url.QueryEscape(artist)
			}
			artistsStr := strings.Join(selectedArtists, ",")

			if len(selectedGenres) > 0 {
				params = append(params, fmt.Sprintf("genre_name=%s", genresStr))
			}

			if len(selectedSubgenres) > 0 {
				params = append(params, fmt.Sprintf("sub_genre_id=%s", subgenreIdsStr))
			}

			if len(selectedArtists) > 0 {
				params = append(params, fmt.Sprintf("artist_name=%s", artistsStr))
			}

			if dateFrom != "" || dateTo != "" {
				params = append(params, fmt.Sprintf("new_release_date=%s:%s", dateFrom, dateTo))
			}

			paramsTotal := strings.Join(params, "&")

			if link.Params != "" {
				paramsTotal = fmt.Sprintf("&%s", paramsTotal)
			} else {
				paramsTotal = fmt.Sprintf("?%s", paramsTotal)
			}

			rawURL += paramsTotal

			app.urls = append(app.urls, rawURL)
			return
		}
	}
}

// selectFromList prints a numbered list and returns the names the user chose plus a back flag.
// Returns nil on Enter & asterisk; back=true if user types b.
func selectFromList(heading string, entries []rankEntry) ([]string, bool) {
	if len(entries) == 0 {
		return nil, false
	}
	fmt.Printf("%s found:\n", heading)
	for i, e := range entries {
		fmt.Printf("  %2d. %-42s %d tracks\n", i+1, e.name, e.count)
	}
	fmt.Print("Select (e.g. 1,3  |  * for all  |  Enter to skip  |  b to go back): ")
	input := strings.TrimSpace(GetLine())

	if input == "b" {
		return nil, true
	}
	if input == "" || input == "*" {
		return nil, false
	}

	var selected []string
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(entries) {
			fmt.Printf("  (ignored invalid selection: %q)\n", part)
			continue
		}
		selected = append(selected, entries[n-1].name)
	}
	return selected, false
}

// normaliseDateFrom accepts "1996", "1996-06", or "1996-06-01" and returns "YYYY-MM-DD" (start of period).
func normaliseDate(input string) string {
	return normaliseDateBound(input, false)
}

// normaliseDateTo resolves to the end of the given year or month.
func normaliseDateTo(input string) string {
	return normaliseDateBound(input, true)
}

func normaliseDateBound(input string, endOfPeriod bool) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	switch len(input) {
	case 4: // "1996"
		if endOfPeriod {
			return input + "-12-31"
		}
		return input + "-01-01"
	case 7: // "1996-06"
		if endOfPeriod {
			return input + "-31" // good enough for string comparison purposes
		}
		return input + "-01"
	default:
		return input
	}
}

func (app *application) search(input string) {
	results, err := app.bp.Search(input)
	if err != nil {
		app.FatalError("beatport", err)
	}
	trackResultsLen := len(results.Tracks)
	releasesResultsLen := len(results.Releases)
	labelsResultsLen := len(results.Labels)
	totalResultsLen := trackResultsLen + releasesResultsLen + labelsResultsLen

	if totalResultsLen == 0 {
		fmt.Println("No results found")
		return
	}

	fmt.Println("Search results:")
	fmt.Println("[ Tracks ]")
	for i, track := range results.Tracks {
		fmt.Printf(
			"%2d. %s - %s (%s) [%s]\n", i+1,
			track.Artists.Display(app.config.ArtistsLimit, app.config.ArtistsShortForm),
			track.Name.String(), track.MixName.String(), track.Length,
		)
	}
	lastTrackNum := trackResultsLen

	fmt.Println("\n[ Releases ]")
	for i, release := range results.Releases {
		fmt.Printf(
			"%2d. %s - %s [%s]\n", i+lastTrackNum+1,
			release.Artists.Display(app.config.ArtistsLimit, app.config.ArtistsShortForm),
			release.Name.String(), release.Label.Name,
		)
	}
	lastReleaseNum := lastTrackNum + releasesResultsLen

	fmt.Println("\n[ Labels ]")
	for i, label := range results.Labels {
		fmt.Printf("%2d. %s\n", i+lastReleaseNum+1, label.Name)
	}
	lastLabelNum := lastReleaseNum + labelsResultsLen

	fmt.Print("Enter the result number(s): ")
	input = GetLine()
	requestedResults := strings.Split(input, " ")

	for _, result := range requestedResults {
		nRes, err := strconv.Atoi(result)
		if err != nil || nRes <= 0 || nRes > totalResultsLen {
			fmt.Printf("invalid result number: %s\n", result)
			continue
		}

		if nRes <= lastTrackNum {
			app.urls = append(app.urls, results.Tracks[nRes-1].URL)
			continue
		}

		if nRes <= lastReleaseNum {
			app.urls = append(app.urls, results.Releases[nRes-1-lastTrackNum].URL)
			continue
		}

		if nRes <= lastLabelNum {
			app.urls = append(app.urls, results.Labels[nRes-1-lastReleaseNum].StoreUrl())
			continue
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
