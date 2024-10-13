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
	TrackLink   LinkType = "tracks"
	ReleaseLink LinkType = "releases"
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

	if len(segments) >= 3 && (segments[0] == "track" || segments[0] == "release") {
		id, err := strconv.ParseInt(segments[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id: %v", err)
		}
		linkType := TrackLink
		if segments[0] == "release" {
			linkType = ReleaseLink
		}
		return &Link{
			Type: linkType,
			ID:   id,
		}, nil
	}

	if len(segments) >= 4 && (segments[2] == "tracks" || segments[2] == "releases") {
		id, err := strconv.ParseInt(segments[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id: %v", err)
		}
		linkType := TrackLink
		if segments[2] == "releases" {
			linkType = ReleaseLink
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
