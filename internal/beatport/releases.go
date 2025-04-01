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
	BPMRange      ReleaseBPMRange `json:"bpm_range"`
	TrackUrls     []string        `json:"tracks"`
	TrackCount    int             `json:"track_count"`
	URL           string          `json:"url"`
	Store         Store           `json:"store"`
}

type ReleaseBPMRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func (r *Release) StoreUrl() string {
	return storeUrl(r.ID, "release", r.Slug, r.Store)
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
	response.Store = b.store
	return response, nil
}

func (b *Beatport) GetReleaseTracks(id int64, page int, params string) (*Paginated[Track], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/releases/%d/tracks/?page=%d&%s", id, page, params),
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
	for i := range response.Results {
		response.Results[i].Store = b.store
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

func (r *Release) DirectoryName(n NamingPreferences) string {
	artistsString := r.Artists.Display(n.ArtistsLimit, n.ArtistsShortForm)
	remixersString := r.Remixers.Display(n.ArtistsLimit, n.ArtistsShortForm)

	templateValues := map[string]string{
		"id":             strconv.Itoa(int(r.ID)),
		"name":           SanitizeForPath(r.Name.String()),
		"slug":           r.Slug,
		"artists":        SanitizeForPath(artistsString),
		"remixers":       SanitizeForPath(remixersString),
		"date":           r.Date,
		"year":           r.Year(),
		"track_count":    NumberWithPadding(r.TrackCount, r.TrackCount, n.TrackNumberPadding),
		"bpm_range":      fmt.Sprintf("%d-%d", r.BPMRange.Min, r.BPMRange.Max),
		"catalog_number": SanitizeForPath(r.CatalogNumber.String()),
		"upc":            r.UPC,
		"label":          SanitizeForPath(r.Label.Name),
	}
	directoryName := ParseTemplate(n.Template, templateValues)
	return SanitizePath(directoryName, n.Whitespace)
}
