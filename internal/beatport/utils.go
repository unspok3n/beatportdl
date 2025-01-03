package beatport

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type LinkType string

var (
	TrackLink    LinkType = "tracks"
	ReleaseLink  LinkType = "releases"
	PlaylistLink LinkType = "playlists"
	ChartLink    LinkType = "charts"
	LabelLink    LinkType = "labels"
	ArtistLink   LinkType = "artists"
)

type Link struct {
	Type   LinkType
	ID     int64
	Params string
}

var (
	ErrInvalidUrl = errors.New("invalid url")
)

func (b *Beatport) ParseUrl(inputURL string) (*Link, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return nil, err
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	segmentsLength := len(segments)
	var link Link

	if segmentsLength == 0 {
		return nil, ErrInvalidUrl
	}

	if segmentsLength > 1 && len(segments[0]) == 2 {
		segments = segments[1:]
		segmentsLength--

		if segments[0] == "catalog" {
			segments = segments[1:]
			segmentsLength--
		}
	}

	var idSegment int

	switch segments[0] {
	case "track":
		idSegment = 2
		link.Type = TrackLink
	case "release":
		idSegment = 2
		link.Type = ReleaseLink
	case "library":
		switch segments[1] {
		case "playlists":
			idSegment = 2
			link.Type = PlaylistLink
		default:
			return nil, fmt.Errorf("invalid link type: %s/%s", segments[0], segments[1])
		}
	case "playlists":
		idSegment = 2
		link.Type = PlaylistLink
	case "chart":
		idSegment = 2
		link.Type = ChartLink
	case "label":
		idSegment = 2
		link.Type = LabelLink
	case "artist":
		idSegment = 2
		link.Type = ArtistLink

	case "tracks":
		idSegment = 1
		link.Type = TrackLink
	case "releases":
		idSegment = 1
		link.Type = ReleaseLink
	default:
		return nil, ErrInvalidUrl
	}

	if idSegment+1 > segmentsLength {
		return nil, ErrInvalidUrl
	}

	link.ID, err = strconv.ParseInt(segments[idSegment], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %v", err)
	}

	link.Params = u.RawQuery

	return &link, nil
}

func ParseTemplate(template string, values map[string]string) string {
	re := regexp.MustCompile(`\{(\w+)}`)
	result := re.ReplaceAllStringFunc(template, func(placeholder string) string {
		key := strings.Trim(placeholder, "{}")
		if value, found := values[key]; found {
			return value
		}
		return placeholder
	})
	return result
}
