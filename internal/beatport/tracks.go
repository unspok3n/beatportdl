package beatport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Track struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	MixName  string   `json:"mix_name"`
	Number   int      `json:"number"`
	Key      TrackKey `json:"key"`
	BPM      int      `json:"bpm"`
	Genre    Genre    `json:"genre"`
	ISRC     string   `json:"isrc"`
	Length   string   `json:"length"`
	Artists  []Artist `json:"artists"`
	Remixers []Artist `json:"remixers"`
	Release  Release  `json:"release"`
	URL      string   `json:"url"`
}

type TrackKey struct {
	Name string `json:"name"`
}

type Genre struct {
	Name string `json:"name"`
}

type TrackStream struct {
	Location      string `json:"location"`
	StreamQuality string `json:"stream_quality"`
}

func (t *Track) ArtistsDisplay(aType ArtistType) string {
	var artistNames []string
	var artists []Artist
	if aType != ArtistTypeMain {
		artists = t.Remixers
	} else {
		artists = t.Artists
	}
	for _, artist := range artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistsString := strings.Join(artistNames, ", ")
	return artistsString
}

func (t *Track) Filename(template string, whitespace string) string {
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">"}

	artistsString := t.ArtistsDisplay(ArtistTypeMain)
	remixersString := t.ArtistsDisplay(ArtistTypeRemixers)

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

	if whitespace != "" {
		fileName = strings.Replace(fileName, " ", whitespace, -1)
	}

	return fileName
}

func (b *Beatport) GetTrack(id int64) (*Track, error) {
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
	response := &Track{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) DownloadTrack(id int64, quality string) (*TrackStream, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf(
			"/catalog/tracks/%d/download/?quality=%s",
			id,
			url.QueryEscape(quality),
		),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &TrackStream{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}
