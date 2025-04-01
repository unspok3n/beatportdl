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
	LengthMs    Duration        `json:"length_ms"`
	Artists     Artists         `json:"artists"`
	Remixers    Artists         `json:"remixers"`
	PublishDate string          `json:"publish_date"`
	Release     Release         `json:"release"`
	URL         string          `json:"url"`
	Store       Store           `json:"store"`
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
	return storeUrl(t.ID, "track", t.Slug, t.Store)
}

func (t *Track) GenreWithSubgenre(separator string) string {
	if t.Subgenre != nil {
		return fmt.Sprintf("%s %s %s", t.Genre.Name, separator, t.Subgenre.Name)
	}
	return t.Genre.Name
}

func (t *Track) SubgenreOrGenre() string {
	if t.Subgenre != nil {
		return t.Subgenre.Name
	}
	return t.Genre.Name
}

func (t *Track) Filename(n NamingPreferences) string {
	artistsString := t.Artists.Display(n.ArtistsLimit, n.ArtistsShortForm)
	remixersString := t.Remixers.Display(n.ArtistsLimit, n.ArtistsShortForm)
	subgenre := ""
	if t.Subgenre != nil {
		subgenre = t.Subgenre.Name
	}

	templateValues := map[string]string{
		"id":                  strconv.Itoa(int(t.ID)),
		"name":                SanitizeForPath(t.Name.String()),
		"slug":                t.Slug,
		"mix_name":            SanitizeForPath(t.MixName.String()),
		"artists":             SanitizeForPath(artistsString),
		"remixers":            SanitizeForPath(remixersString),
		"number":              NumberWithPadding(t.Number, t.Release.TrackCount, n.TrackNumberPadding),
		"length":              t.LengthMs.Display(),
		"key":                 t.Key.Display(n.KeySystem),
		"bpm":                 strconv.Itoa(t.BPM),
		"genre":               SanitizeForPath(t.Genre.Name),
		"subgenre":            SanitizeForPath(subgenre),
		"genre_with_subgenre": SanitizeForPath(t.GenreWithSubgenre("-")),
		"subgenre_or_genre":   SanitizeForPath(t.SubgenreOrGenre()),
		"isrc":                t.ISRC,
		"label":               SanitizeForPath(t.Release.Label.Name),
	}
	fileName := ParseTemplate(n.Template, templateValues)
	return SanitizePath(fileName, n.Whitespace)
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
	response.Store = b.store
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
