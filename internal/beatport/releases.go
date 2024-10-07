package beatport

import (
	"encoding/json"
	"fmt"
)

type BeatportRelease struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	Artists   []BeatportArtist `json:"artists"`
	TrackUrls []string         `json:"tracks"`
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
