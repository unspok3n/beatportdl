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
	Date          string   `json:"new_release_date"`
	Image         Image    `json:"image"`
	TrackUrls     []string `json:"tracks"`
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

func (r *Release) DirectoryName(template string, whitespace string) string {
	var artistNames []string
	var remixerNames []string
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">", "."}

	for _, artist := range r.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistsString := strings.Join(artistNames, ", ")

	for _, artist := range r.Remixers {
		remixerNames = append(remixerNames, artist.Name)
	}
	remixersString := strings.Join(remixerNames, ", ")

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
