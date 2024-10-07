package beatport

import (
	"encoding/json"
	"fmt"
	"strings"
)

type BeatportTrack struct {
	ID      int64            `json:"id"`
	Name    string           `json:"name"`
	MixName string           `json:"mix_name"`
	Number  int              `json:"number"`
	Artists []BeatportArtist `json:"artists"`
}

type BeatportTrackStream struct {
	Location      string `json:"location"`
	StreamQuality string `json:"stream_quality"`
}

func (t *BeatportTrack) Filename() string {
	var artistNames []string
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">", "."}

	for _, artist := range t.Artists {
		artistNames = append(artistNames, artist.Name)
	}

	artistsString := strings.Join(artistNames, ", ")

	for _, char := range charsToRemove {
		artistsString = strings.Replace(artistsString, char, "", -1)
		t.Name = strings.Replace(t.Name, char, "", -1)
		t.MixName = strings.Replace(t.MixName, char, "", -1)
	}

	filename := fmt.Sprintf(
		"%s - %s (%s).flac",
		artistsString,
		t.Name,
		t.MixName)
	return filename
}

func (b *Beatport) GetTrack(id int64) (*BeatportTrack, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/tracks/%d/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &BeatportTrack{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) DownloadTrack(id int64) (*BeatportTrackStream, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/tracks/%d/download/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &BeatportTrackStream{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}
