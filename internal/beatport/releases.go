package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type BeatportRelease struct {
	ID            int64            `json:"id"`
	Name          string           `json:"name"`
	Artists       []BeatportArtist `json:"artists"`
	Remixers      []BeatportArtist `json:"remixers"`
	CatalogNumber string           `json:"catalog_number"`
	Date          string           `json:"new_release_date"`
	TrackUrls     []string         `json:"tracks"`
}

func (b *Beatport) GetRelease(id int64) (*BeatportRelease, error) {
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
	response := &BeatportRelease{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (r *BeatportRelease) DirectoryName(template string) string {
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

	return directoryName
}
