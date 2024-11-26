package beatport

import (
	"encoding/json"
	"fmt"
)

type Chart struct {
	Name  string `json:"name"`
	Image Image  `json:"image"`
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

func (b *Beatport) GetChartTracks(id int64, page int) (*Paginated[Track], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/charts/%d/tracks/?page=%d", id, page),
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
