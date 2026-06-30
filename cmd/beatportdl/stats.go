package main

import (
	"sort"
	"unspok3n/beatportdl/internal/beatport"
)

type entityStats struct {
	genres      map[string]int
	subgenres   map[string]int
	artists     map[string]int
	subgenreIds map[string]int64
	total       int
}

type rankEntry struct {
	name  string
	count int
}

func rankMap(m map[string]int) []rankEntry {
	entries := make([]rankEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, rankEntry{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].name < entries[j].name
	})
	return entries
}

func newEntityStats(total int, facets *beatport.Facets) *entityStats {
	stats := &entityStats{
		genres:      make(map[string]int),
		subgenres:   make(map[string]int),
		artists:     make(map[string]int),
		subgenreIds: make(map[string]int64),
	}

	stats.total = total

	genres, exists := facets.Fields["genre"]
	if exists && len(genres) > 0 {
		for _, genre := range genres {
			stats.genres[genre.Name] = genre.Count
		}
	}

	subGenres, exists := facets.Fields["sub_genre"]
	if exists && len(subGenres) > 0 {
		for _, subgenre := range subGenres {
			stats.subgenres[subgenre.Name] = subgenre.Count
			stats.subgenreIds[subgenre.Name] = subgenre.ID
		}
	}

	artists, exists := facets.Fields["artists"]
	if exists && len(artists) > 0 {
		for _, artist := range artists {
			stats.artists[artist.Name] = artist.Count
		}
	}

	return stats
}
