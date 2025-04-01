package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Chart struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	TrackCount  int         `json:"track_count"`
	Person      ChartPerson `json:"person"`
	Genres      []Genre     `json:"genres"`
	AddDate     time.Time   `json:"add_date"`
	ChangeDate  time.Time   `json:"change_date"`
	PublishDate time.Time   `json:"publish_date"`
	Image       Image       `json:"image"`
}

type ChartPerson struct {
	OwnerName string `json:"owner_name"`
	OwnerSlug string `json:"owner_slug"`
}

func (c *Chart) DirectoryName(n NamingPreferences) string {
	var firstGenre string
	if len(c.Genres) > 0 {
		firstGenre = c.Genres[0].Name
	}
	templateValues := map[string]string{
		"id":             strconv.Itoa(int(c.ID)),
		"name":           SanitizeForPath(c.Name),
		"slug":           c.Slug,
		"first_genre":    SanitizeForPath(firstGenre),
		"track_count":    NumberWithPadding(c.TrackCount, c.TrackCount, n.TrackNumberPadding),
		"creator":        SanitizeForPath(c.Person.OwnerName),
		"created_date":   c.AddDate.Format("2006-01-02"),
		"published_date": c.PublishDate.Format("2006-01-02"),
		"updated_date":   c.ChangeDate.Format("2006-01-02"),
	}
	directoryName := ParseTemplate(n.Template, templateValues)
	return SanitizePath(directoryName, n.Whitespace)
}

func (b *Beatport) GetChart(id int64) (*Chart, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/charts/%d/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &Chart{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) GetChartTracks(id int64, page int, params string) (*Paginated[Track], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/charts/%d/tracks/?page=%d&%s", id, page, params),
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
