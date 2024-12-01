package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Release struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Artists       []Artist `json:"artists"`
	Remixers      []Artist `json:"remixers"`
	CatalogNumber string   `json:"catalog_number"`
	Label         Label    `json:"label"`
	Date          string   `json:"new_release_date"`
	Image         Image    `json:"image"`
	TrackUrls     []string `json:"tracks"`
	URL           string   `json:"url"`
}

func (r *Release) ArtistsDisplay(aType ArtistType, limit int, shortForm string) string {
	var artistNames []string
	var artists []Artist
	if aType != ArtistTypeMain {
		artists = r.Remixers
	} else {
		artists = r.Artists
	}
	if shortForm != "" && len(artists) > limit {
		return shortForm
	}
	for _, artist := range artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistsString := strings.Join(artistNames, ", ")
	return artistsString
}

func (b *Beatport) GetRelease(id int64) (*Release, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/releases/%d/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &Release{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (r *Release) DirectoryName(template string, whitespace string, aLimit int, aShortForm string) string {
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">", "."}

	artistsString := r.ArtistsDisplay(ArtistTypeMain, aLimit, aShortForm)
	remixersString := r.ArtistsDisplay(ArtistTypeRemixers, aLimit, aShortForm)

	templateValues := map[string]string{
		"id":             strconv.Itoa(int(r.ID)),
		"name":           r.Name,
		"artists":        artistsString,
		"remixers":       remixersString,
		"date":           r.Date,
		"catalog_number": r.CatalogNumber,
	}
	directoryName := ParseTemplate(template, templateValues)

	for _, char := range charsToRemove {
		directoryName = strings.Replace(directoryName, char, "", -1)
	}

	if len(directoryName) > 250 {
		directoryName = directoryName[:250]
	}

	if whitespace != "" {
		directoryName = strings.Replace(
			directoryName,
			" ",
			whitespace,
			-1,
		)
	}

	return directoryName
}
