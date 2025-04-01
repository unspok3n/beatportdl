package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Artist struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Artists []Artist

func (a *Artist) DirectoryName(n NamingPreferences) string {
	templateValues := map[string]string{
		"id":   strconv.Itoa(int(a.ID)),
		"name": SanitizeForPath(a.Name),
		"slug": a.Slug,
	}
	directoryName := ParseTemplate(n.Template, templateValues)
	return SanitizePath(directoryName, n.Whitespace)
}

func (a *Artists) Display(limit int, shortForm string) string {
	var artistNames []string
	if shortForm != "" && len(*a) > limit {
		return shortForm
	}
	for _, artist := range *a {
		artistNames = append(artistNames, artist.Name)
	}
	artistsString := strings.Join(artistNames, ", ")
	return artistsString
}

func (b *Beatport) GetArtist(id int64) (*Artist, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/artists/%d/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &Artist{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) GetArtistTracks(id int64, page int, params string) (*Paginated[Track], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/artists/%d/tracks/?page=%d&%s", id, page, params),
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
