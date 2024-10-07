package beatport

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type BeatportLinkType string

var (
	BeatportTrackLink   BeatportLinkType = "tracks"
	BeatportReleaseLink BeatportLinkType = "releases"
)

type BeatportLink struct {
	Type BeatportLinkType
	ID   int64
}

var (
	ErrInvalidUrl = errors.New("invalid url")
)

func (b *Beatport) ParseUrl(inputURL string) (*BeatportLink, error) {
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
		linkType := BeatportTrackLink
		if segments[0] == "release" {
			linkType = BeatportReleaseLink
		}
		return &BeatportLink{
			Type: linkType,
			ID:   id,
		}, nil
	}

	if len(segments) >= 4 && (segments[2] == "tracks" || segments[2] == "releases") {
		id, err := strconv.ParseInt(segments[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id: %v", err)
		}
		linkType := BeatportTrackLink
		if segments[2] == "releases" {
			linkType = BeatportReleaseLink
		}
		return &BeatportLink{
			Type: linkType,
			ID:   id,
		}, nil
	}

	return nil, ErrInvalidUrl
}
