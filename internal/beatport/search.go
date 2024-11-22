package beatport

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type SearchResults struct {
	Tracks   []Track   `json:"tracks"`
	Releases []Release `json:"releases"`
}

func (b *Beatport) Search(query string) (*SearchResults, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/search/?q=%s&order_by=-publish_date&is_available_for_streaming=true", url.QueryEscape(query)),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &SearchResults{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}
