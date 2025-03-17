package config

import (
	"fmt"
	"unspok3n/beatportdl/internal/validator"
)

func ValidateTagMappings(m map[string]map[string]string) error {
	for format, mappings := range m {
		if !validator.PermittedValue(format, SupportedTagMappingFormats...) {
			return fmt.Errorf("invalid tag mapping format '%s'", format)
		}

		for field := range mappings {
			if !validator.PermittedValue(field, SupportedTagMappingFields...) {
				return fmt.Errorf("invalid tag mapping field '%s'", field)
			}
		}
	}
	return nil
}

var (
	SupportedTagMappingFormats = []string{
		"flac",
		"m4a",
	}

	SupportedTagMappingFields = []string{
		"track_id",
		"track_url",
		"track_name",
		"track_artists",
		"track_remixers",
		"track_artists_limited",
		"track_remixers_limited",
		"track_number",
		"track_number_with_padding",
		"track_number_with_total",
		"track_genre",
		"track_subgenre",
		"track_genre_with_subgenre",
		"track_subgenre_or_genre",
		"track_key",
		"track_bpm",
		"track_isrc",

		"release_id",
		"release_url",
		"release_name",
		"release_artists",
		"release_remixers",
		"release_artists_limited",
		"release_remixers_limited",
		"release_date",
		"release_year",
		"release_track_count",
		"release_track_count_with_padding",
		"release_catalog_number",
		"release_upc",
		"release_label",
		"release_label_url",
	}

	DefaultTagMappings = map[string]map[string]string{
		"flac": {
			"track_name":              "TITLE",
			"track_artists":           "ARTIST",
			"track_number":            "TRACKNUMBER",
			"track_subgenre_or_genre": "GENRE",
			"track_key":               "KEY",
			"track_bpm":               "BPM",
			"track_isrc":              "ISRC",

			"release_name":           "ALBUM",
			"release_artists":        "ALBUMARTIST",
			"release_date":           "DATE",
			"release_track_count":    "TOTALTRACKS",
			"release_catalog_number": "CATALOGNUMBER",
			"release_label":          "LABEL",
		},
		"m4a": {
			"track_name":    "TITLE",
			"track_artists": "ARTIST",
			"track_number":  "TRACKNUMBER",
			"track_genre":   "GENRE",
			"track_key":     "KEY",
			"track_bpm":     "BPM",
			"track_isrc":    "ISRC",

			"release_name":           "ALBUM",
			"release_artists":        "ALBUMARTIST",
			"release_date":           "DATE",
			"release_track_count":    "TOTALTRACKS",
			"release_catalog_number": "CATALOGNUMBER",
			"release_label":          "LABEL",
		},
	}
)
