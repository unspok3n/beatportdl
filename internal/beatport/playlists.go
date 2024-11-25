package beatport

import (
	"encoding/json"
	"fmt"
)

type PlaylistItem struct {
	ID       int64 `json:"id"`
	Position int64 `json:"position"`
	Track    Track `json:"track"`
}

func (b *Beatport) GetPlaylistItems(id int64, page int) (*Paginated[PlaylistItem], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/playlists/%d/tracks/?page=%d", id, page),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var response Paginated[PlaylistItem]
	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}
