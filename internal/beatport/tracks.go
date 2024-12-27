package beatport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type Track struct {
	ID          int64           `json:"id"`
	Name        SanitizedString `json:"name"`
	MixName     SanitizedString `json:"mix_name"`
	Slug        string          `json:"slug"`
	Number      int             `json:"number"`
	Key         Key             `json:"key"`
	BPM         int             `json:"bpm"`
	Genre       Genre           `json:"genre"`
	Subgenre    *Genre          `json:"sub_genre"`
	ISRC        string          `json:"isrc"`
	Length      string          `json:"length"`
	LengthMs    int             `json:"length_ms"`
	Artists     Artists         `json:"artists"`
	Remixers    Artists         `json:"remixers"`
	PublishDate string          `json:"publish_date"`
	Release     Release         `json:"release"`
	URL         string          `json:"url"`
}

type TrackDownload struct {
	Location      string `json:"location"`
	StreamQuality string `json:"stream_quality"`
}

type TrackStream struct {
	Url           string `json:"stream_url"`
	SampleStartMs int    `json:"sample_start_ms"`
	SampleEndMs   int    `json:"sample_end_ms"`
}

func (t *Track) StoreUrl() string {
	return fmt.Sprintf("https://www.beatport.com/track/%s/%d", t.Slug, t.ID)
}

func (t *Track) Filename(template string, whitespace string, aLimit int, aShortForm string, keySystem string) string {
	artistsString := t.Artists.Display(aLimit, aShortForm)
	remixersString := t.Remixers.Display(aLimit, aShortForm)

	templateValues := map[string]string{
		"id":       strconv.Itoa(int(t.ID)),
		"name":     t.Name.String(),
		"mix_name": t.MixName.String(),
		"artists":  artistsString,
		"remixers": remixersString,
		"number":   fmt.Sprintf("%02d", t.Number),
		"key":      t.Key.Display(keySystem),
		"bpm":      strconv.Itoa(t.BPM),
		"genre":    t.Genre.Name,
		"isrc":     t.ISRC,
	}
	fileName := ParseTemplate(template, templateValues)
	return SanitizePath(fileName, whitespace)
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

func (b *Beatport) DownloadTrack(id int64, quality string) (*TrackDownload, error) {
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
	response := &TrackDownload{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) StreamTrack(id int64) (*TrackStream, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf(
			"/catalog/tracks/%d/stream/",
			id,
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
