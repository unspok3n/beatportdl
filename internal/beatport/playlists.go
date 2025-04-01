package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Playlist struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Genres      []string  `json:"genres"`
	TrackCount  int       `json:"track_count"`
	BPMRange    []*int    `json:"bpm_range"`
	LengthMs    Duration  `json:"length_ms"`
	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

type PlaylistItem struct {
	ID       int64 `json:"id"`
	Position int   `json:"position"`
	Track    Track `json:"track"`
}

func (p *Playlist) DirectoryName(n NamingPreferences) string {
	var firstGenre string
	var bpmRange string

	if len(p.Genres) > 0 {
		firstGenre = p.Genres[0]
	}

	if len(p.BPMRange) > 0 && p.BPMRange[0] != nil && p.BPMRange[1] != nil {
		bpmRange = fmt.Sprintf("%d-%d", *p.BPMRange[0], *p.BPMRange[1])
	}

	templateValues := map[string]string{
		"id":           strconv.Itoa(int(p.ID)),
		"name":         SanitizeForPath(p.Name),
		"first_genre":  SanitizeForPath(firstGenre),
		"track_count":  NumberWithPadding(p.TrackCount, p.TrackCount, n.TrackNumberPadding),
		"bpm_range":    bpmRange,
		"length":       p.LengthMs.Display(),
		"created_date": p.CreatedDate.Format("2006-01-02"),
		"updated_date": p.UpdatedDate.Format("2006-01-02"),
	}
	directoryName := ParseTemplate(n.Template, templateValues)
	return SanitizePath(directoryName, n.Whitespace)
}

func (b *Beatport) GetPlaylist(id int64) (*Playlist, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/playlists/%d/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &Playlist{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) GetPlaylistItems(id int64, page int, params string) (*Paginated[PlaylistItem], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/playlists/%d/tracks/?page=%d&%s", id, page, params),
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
	for i := range response.Results {
		response.Results[i].Track.Store = b.store
	}
	return &response, nil
}
