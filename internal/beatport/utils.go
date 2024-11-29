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
)

type Link struct {
	Type LinkType
	ID   int64
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

	if segmentsLength == 4 && len(segments[0]) == 2 && segments[0] != "v4" {
		segments = segments[1:]
		segmentsLength--
	}

	if segmentsLength == 3 {
		id, err := strconv.ParseInt(segments[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id: %v", err)
		}
		var linkType LinkType
		switch segments[0] {
		case "track":
			linkType = TrackLink
		case "release":
			linkType = ReleaseLink
		case "library":
			switch segments[1] {
			case "playlists":
				linkType = PlaylistLink
			default:
				return nil, fmt.Errorf("invalid link type: %s/%s", segments[0], segments[1])
			}
		case "playlists":
			linkType = PlaylistLink
		case "chart":
			linkType = ChartLink
		default:
			return nil, fmt.Errorf("invalid link type: %s", segments[0])
		}
		return &Link{
			Type: linkType,
			ID:   id,
		}, nil
	}

	if segmentsLength == 4 {
		id, err := strconv.ParseInt(segments[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id: %v", err)
		}
		var linkType LinkType
		switch segments[2] {
		case "tracks":
			linkType = TrackLink
		case "releases":
			linkType = ReleaseLink
		default:
			return nil, fmt.Errorf("invalid link type: %s", segments[2])
		}
		return &Link{
			Type: linkType,
			ID:   id,
		}, nil
	}

	return nil, ErrInvalidUrl
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
