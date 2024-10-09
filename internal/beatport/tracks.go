package beatport

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type BeatportTrack struct {
	ID       int64            `json:"id"`
	Name     string           `json:"name"`
	MixName  string           `json:"mix_name"`
	Number   int              `json:"number"`
	Key      BeatportTrackKey `json:"key"`
	BPM      int              `json:"bpm"`
	Genre    BeatportGenre    `json:"genre"`
	ISRC     string           `json:"isrc"`
	Artists  []BeatportArtist `json:"artists"`
	Remixers []BeatportArtist `json:"remixers"`
	Release  BeatportRelease  `json:"release"`
}

type BeatportTrackKey struct {
	Name string `json:"name"`
}

type BeatportGenre struct {
	Name string `json:"name"`
}

type BeatportTrackStream struct {
	Location      string `json:"location"`
	StreamQuality string `json:"stream_quality"`
}

func (t *BeatportTrack) Filename(template string) string {
	var artistNames []string
	var remixerNames []string
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">"}

	for _, artist := range t.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistsString := strings.Join(artistNames, ", ")

	for _, artist := range t.Remixers {
		remixerNames = append(remixerNames, artist.Name)
	}
	remixersString := strings.Join(remixerNames, ", ")

	templateValues := map[string]string{
		"id":       strconv.Itoa(int(t.ID)),
		"name":     t.Name,
		"mix_name": t.MixName,
		"artists":  artistsString,
		"remixers": remixersString,
		"number":   fmt.Sprintf("%02d", t.Number),
		"key":      t.Key.Name,
		"bpm":      strconv.Itoa(t.BPM),
		"genre":    t.Genre.Name,
		"isrc":     t.ISRC,
	}
	fileName := ParseTemplate(template, templateValues)

	for _, char := range charsToRemove {
		fileName = strings.Replace(fileName, char, "", -1)
	}

	if len(fileName) > 250 {
		fileName = fileName[:250]
	}

	return fileName + ".flac"
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
