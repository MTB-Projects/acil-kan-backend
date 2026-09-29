// Package hospital serves the hospital directory used for autocomplete.
//
// The data (hospitals.json) is generated from OpenStreetMap by
// scripts/osm/build.js and embedded in the binary; refreshing it means
// regenerating the file and redeploying. Data © OpenStreetMap contributors (ODbL).
package hospital

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"acilkan.backend/pkg/textnorm"
)

//go:embed hospitals.json
var embedded []byte

// Hospital is one entry of the directory.
type Hospital struct {
	ID       string  `json:"id"` // OSM id, e.g. "w123456"
	Name     string  `json:"name"`
	City     string  `json:"city"`
	District string  `json:"district"`
	Address  string  `json:"address,omitempty"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type entry struct {
	h        Hospital
	nameFold string
	words    []string // folded words of name and district
	city     string   // folded
	district string   // folded
}

// Directory is an in-memory, read-only hospital index.
type Directory struct {
	entries []entry
}

// Load parses the embedded hospital list.
func Load() (*Directory, error) {
	return Parse(embedded)
}

// Parse builds a directory from JSON (a list of Hospital).
func Parse(data []byte) (*Directory, error) {
	var list []Hospital
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("hospital directory: %w", err)
	}
	d := &Directory{entries: make([]entry, 0, len(list))}
	for _, h := range list {
		d.entries = append(d.entries, entry{
			h:        h,
			nameFold: textnorm.Fold(h.Name),
			words:    append(textnorm.Words(h.Name), textnorm.Words(h.District)...),
			city:     textnorm.Fold(h.City),
			district: textnorm.Fold(h.District),
		})
	}
	sort.SliceStable(d.entries, func(i, j int) bool {
		return d.entries[i].nameFold < d.entries[j].nameFold
	})
	return d, nil
}

// Len returns the number of hospitals.
func (d *Directory) Len() int { return len(d.entries) }

// Search returns up to limit hospitals in city (and district, if given) whose
// name matches q. Every word of q must be the prefix of a word in the hospital's
// name or district, ignoring case and Turkish letters ("sehir" finds "Şehir").
// Names that start with q come first. An empty q lists hospitals alphabetically.
func (d *Directory) Search(city, district, q string, limit int) []Hospital {
	cityFold := strings.TrimSpace(textnorm.Fold(city))
	districtFold := strings.TrimSpace(textnorm.Fold(district))
	queryWords := textnorm.Words(q)
	queryFold := strings.Join(queryWords, " ")

	type scored struct {
		h     Hospital
		score int
	}
	var matches []scored
	for _, e := range d.entries {
		if cityFold != "" && e.city != cityFold {
			continue
		}
		if districtFold != "" && e.district != districtFold {
			continue
		}
		if !allPrefixed(queryWords, e.words) {
			continue
		}
		score := 2
		if queryFold == "" || strings.HasPrefix(e.nameFold, queryFold) {
			score = 0
		} else if len(e.words) > 0 && strings.HasPrefix(e.words[0], queryWords[0]) {
			score = 1
		}
		matches = append(matches, scored{e.h, score})
	}

	// entries are already sorted by name, so a stable sort keeps names ordered within a score
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score < matches[j].score })

	if limit <= 0 || limit > len(matches) {
		limit = len(matches)
	}
	out := make([]Hospital, 0, limit)
	for _, m := range matches[:limit] {
		out = append(out, m.h)
	}
	return out
}

func allPrefixed(query, words []string) bool {
	for _, q := range query {
		found := false
		for _, w := range words {
			if strings.HasPrefix(w, q) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
