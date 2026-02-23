package cmd

import "testing"

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "b", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
	}

	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuggestID(t *testing.T) {
	candidates := []string{
		"github.com/owner/repo@main",
		"github.com/owner/other@develop",
		"github.com/foo/bar@main",
	}

	tests := []struct {
		input string
		want  string
	}{
		// Close match — 1 char off
		{"github.com/owner/repo@maim", "github.com/owner/repo@main"},
		// Too far — totally different
		{"something-completely-different", ""},
		// Exact match
		{"github.com/foo/bar@main", "github.com/foo/bar@main"},
		// Empty input
		{"", ""},
	}

	for _, tt := range tests {
		got := suggestID(tt.input, candidates)
		if got != tt.want {
			t.Errorf("suggestID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Empty candidates
	if got := suggestID("anything", nil); got != "" {
		t.Errorf("suggestID with nil candidates = %q, want empty", got)
	}
}
