package cmd

import (
	"errors"
	"fmt"

	"github.com/wu-json/pickpocket/internal/cache"
	"github.com/wu-json/pickpocket/internal/giturl"
	"github.com/wu-json/pickpocket/internal/pickfile"
)

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			m := ins
			if del < m {
				m = del
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// suggestID returns the closest match from candidates if within ~30% edit distance,
// or an empty string if nothing is close enough.
func suggestID(input string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	best := ""
	bestDist := len(input) + 1

	for _, c := range candidates {
		d := levenshtein(input, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}

	// Threshold: ~30% of the longer string's length, minimum 1
	maxLen := len(input)
	if len(best) > maxLen {
		maxLen = len(best)
	}
	threshold := maxLen * 3 / 10
	if threshold < 1 {
		threshold = 1
	}

	if bestDist <= threshold {
		return best
	}
	return ""
}

// collectPickIDs builds a list of cache IDs from a Pickfile.
func collectPickIDs(pf *pickfile.Pickfile) []string {
	var ids []string
	for _, p := range pf.Picks {
		pu, err := giturl.Parse(p.URL)
		if err != nil {
			continue
		}
		ids = append(ids, pu.CacheID(p.Branch))
	}
	return ids
}

// collectCacheIDs builds a list of IDs from a cache Index.
func collectCacheIDs(idx *cache.Index) []string {
	ids := make([]string, len(idx.Repos))
	for i, r := range idx.Repos {
		ids[i] = r.ID
	}
	return ids
}

// findPickByID resolves a pick by its cache ID.
// If no match is found, suggests the closest ID if available.
func findPickByID(pf *pickfile.Pickfile, id string) (*pickfile.Pick, giturl.ParsedURL, error) {
	for i, p := range pf.Picks {
		pu, err := giturl.Parse(p.URL)
		if err != nil {
			continue
		}
		if pu.CacheID(p.Branch) == id {
			return &pf.Picks[i], pu, nil
		}
	}
	msg := fmt.Sprintf("no pick found matching %q", id)
	if s := suggestID(id, collectPickIDs(pf)); s != "" {
		msg += fmt.Sprintf("\n\n  did you mean %s?", s)
	}
	return nil, giturl.ParsedURL{}, errors.New(msg)
}
