package beatport

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Artist struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Artists []Artist

func (a Artist) NameSanitized() string {
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">", "."}
	for _, char := range charsToRemove {
		a.Name = strings.Replace(a.Name, char, "", -1)
	}
	return a.Name
}

func (a Artists) Display(limit int, shortForm string) string {
	var artistNames []string
	if shortForm != "" && len(a) > limit {
		return shortForm
	}
	for _, artist := range a {
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

func (b *Beatport) GetArtistTracks(id int64, page int) (*Paginated[Track], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/artists/%d/tracks/?page=%d", id, page),
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
