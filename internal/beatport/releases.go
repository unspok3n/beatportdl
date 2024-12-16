package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Release struct {
	ID            int64           `json:"id"`
	Name          SanitizedString `json:"name"`
	Slug          string          `json:"slug"`
	Artists       Artists         `json:"artists"`
	Remixers      Artists         `json:"remixers"`
	CatalogNumber SanitizedString `json:"catalog_number"`
	Label         Label           `json:"label"`
	Date          string          `json:"new_release_date"`
	Image         Image           `json:"image"`
	TrackUrls     []string        `json:"tracks"`
	URL           string          `json:"url"`
}

func (r *Release) StoreUrl() string {
	return fmt.Sprintf("https://www.beatport.com/release/%s/%d", r.Slug, r.ID)
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

func (b *Beatport) GetReleaseTracks(id int64, page int) (*Paginated[Track], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/releases/%d/tracks/?page=%d", id, page),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var response Paginated[Track]
	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (r *Release) DirectoryName(template string, whitespace string, aLimit int, aShortForm string) string {
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">", "."}

	artistsString := r.Artists.Display(aLimit, aShortForm)
	remixersString := r.Remixers.Display(aLimit, aShortForm)

	var year string
	dateParsed, err := time.Parse("2006-01-02", r.Date)
	if err == nil {
		year = dateParsed.Format("2006")
	}

	templateValues := map[string]string{
		"id":             strconv.Itoa(int(r.ID)),
		"name":           r.Name.String(),
		"artists":        artistsString,
		"remixers":       remixersString,
		"date":           r.Date,
		"year":           year,
		"catalog_number": r.CatalogNumber.String(),
	}
	directoryName := ParseTemplate(template, templateValues)

	for _, char := range charsToRemove {
		directoryName = strings.Replace(directoryName, char, "", -1)
		directoryName = strings.Join(strings.Fields(directoryName), " ")
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
