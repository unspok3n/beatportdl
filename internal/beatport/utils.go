package beatport

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type SanitizedString string
type Duration int
type NamingPreferences struct {
	Template           string
	Whitespace         string
	ArtistsLimit       int
	ArtistsShortForm   string
	TrackNumberPadding int
	KeySystem          string
}

func (d *Duration) Display() string {
	seconds := *d / 1000
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	remainingSeconds := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d-%02d-%02d", hours, minutes, remainingSeconds)
	}
	return fmt.Sprintf("%02d-%02d", minutes, remainingSeconds)
}

func (s *SanitizedString) UnmarshalJSON(data []byte) error {
	rawValue := string(bytes.Trim(data, `"`))
	r := strings.NewReplacer(
		"\\n", "",
		"\\r", "",
		"\\t", "",
	)
	sanitized := r.Replace(rawValue)
	*s = SanitizedString(strings.Join(strings.Fields(sanitized), " "))
	return nil
}

func (s *SanitizedString) String() string {
	return string(*s)
}

func SanitizeForPath(s string) string {
	r := strings.NewReplacer(
		"\\", "",
		"/", "",
	)
	return strings.Join(strings.Fields(r.Replace(s)), " ")
}

func SanitizePath(name string, whitespace string) string {
	if len(name) > 250 {
		name = name[:250]
	}

	oldnew := []string{
		"<", "",
		">", "",
		":", "",
		"\"", "",
		"|", "",
		"?", "",
		"*", "",
	}

	if whitespace != "" {
		oldnew = append(oldnew, " ", whitespace)
	}

	r := strings.NewReplacer(oldnew...)
	name = r.Replace(name)

	return strings.Join(strings.Fields(name), " ")
}

func NumberWithPadding(value, total, padding int) string {
	if padding == 0 {
		padding = len(strconv.Itoa(total))
	}
	return fmt.Sprintf("%0*d", padding, value)
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

func storeUrl(id int64, entity, slug string, store Store) string {
	var domain string
	switch store {
	default:
		domain = "beatport.com"
	case StoreBeatsource:
		domain = "beatsource.com"
	}
	return fmt.Sprintf("https://www.%s/%s/%s/%d", domain, entity, slug, id)
}
