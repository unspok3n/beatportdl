package beatport

import "strings"

type Image struct {
	ID         int64  `json:"id"`
	URI        string `json:"uri"`
	DynamicURI string `json:"dynamic_uri"`
}

func (i *Image) FormattedUrl(size string) string {
	return strings.Replace(
		i.DynamicURI,
		"{w}x{h}",
		size,
		-1,
	)
}
