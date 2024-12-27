package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Release struct {
	ID            int64           `json:"id"`
	Name          SanitizedString `json:"name"`
	Slug          string          `json:"slug"`
	Artists       Artists         `json:"artists"`
	Remixers      Artists         `json:"remixers"`
	CatalogNumber SanitizedString `json:"catalog_number"`
	UPC           string          `json:"upc"`
	Label         Label           `json:"label"`
	Date          string          `json:"new_release_date"`
	Image         Image           `json:"image"`
	TrackUrls     []string        `json:"tracks"`
	TrackCount    int             `json:"track_count"`
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

func (r *Release) Year() string {
	var year string
	dateParsed, err := time.Parse("2006-01-02", r.Date)
	if err == nil {
		year = dateParsed.Format("2006")
	}
	return year
}

func (r *Release) DirectoryName(template string, whitespace string, aLimit int, aShortForm string) string {
	artistsString := r.Artists.Display(aLimit, aShortForm)
	remixersString := r.Remixers.Display(aLimit, aShortForm)

	templateValues := map[string]string{
		"id":             strconv.Itoa(int(r.ID)),
		"name":           SanitizeForPath(r.Name.String()),
		"artists":        SanitizeForPath(artistsString),
		"remixers":       SanitizeForPath(remixersString),
		"date":           r.Date,
		"year":           r.Year(),
		"catalog_number": r.CatalogNumber.String(),
	}
	directoryName := ParseTemplate(template, templateValues)
	return SanitizePath(directoryName, whitespace)
}
