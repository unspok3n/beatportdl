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

// These regexps and replacers are immutable and reused on hot paths
// (filename and directory templating, JSON unmarshalling), so they are
// compiled once at package initialization instead of on every call.
var (
	templatePlaceholderRegexp = regexp.MustCompile(`\{(\w+)}`)

	sanitizedStringReplacer = strings.NewReplacer(
		"\\n", "",
		"\\r", "",
		"\\t", "",
	)

	sanitizeForPathReplacer = strings.NewReplacer(
		"\\", "",
		"/", "",
	)

	sanitizePathReplacer = strings.NewReplacer(
		"<", "",
		">", "",
		":", "",
		"\"", "",
		"|", "",
		"?", "",
		"*", "",
	)
)

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
	sanitized := sanitizedStringReplacer.Replace(rawValue)
	*s = SanitizedString(strings.Join(strings.Fields(sanitized), " "))
	return nil
}

func (s *SanitizedString) String() string {
	return string(*s)
}

func SanitizeForPath(s string) string {
	return strings.Join(strings.Fields(sanitizeForPathReplacer.Replace(s)), " ")
}

func SanitizePath(name string, whitespace string) string {
	if len(name) > 250 {
		name = name[:250]
	}

	name = sanitizePathReplacer.Replace(name)
	if whitespace != "" {
		name = strings.ReplaceAll(name, " ", whitespace)
	}

	return strings.Join(strings.Fields(name), " ")
}

func NumberWithPadding(value, total, padding int) string {
	if padding == 0 {
		padding = len(strconv.Itoa(total))
	}
	return fmt.Sprintf("%0*d", padding, value)
}

func ParseTemplate(template string, values map[string]string) string {
	result := templatePlaceholderRegexp.ReplaceAllStringFunc(template, func(placeholder string) string {
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
