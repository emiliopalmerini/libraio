package commands

import (
	"testing"

	"libraio/internal/domain"
)

func TestFuzzyScore(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		query     string
		wantScore int
		wantMin   int // use this for relative comparisons
	}{
		{
			name:      "exact match",
			target:    "Theatre",
			query:     "Theatre",
			wantScore: 150, // 100 for contains + 50 for prefix
		},
		{
			name:      "prefix match",
			target:    "Theatre Season",
			query:     "Theatre",
			wantScore: 150, // 100 for contains + 50 for prefix
		},
		{
			name:      "substring match",
			target:    "My Theatre",
			query:     "Theatre",
			wantScore: 100, // contains only
		},
		{
			name:    "fuzzy match all chars at start",
			target:  "Theatre",
			query:   "the",
			wantMin: 100, // should be high due to prefix
		},
		{
			name:      "no match",
			target:    "Theatre",
			query:     "xyz",
			wantScore: 0,
		},
		{
			name:      "empty query",
			target:    "Theatre",
			query:     "",
			wantScore: 0,
		},
		{
			name:    "case insensitive",
			target:  "THEATRE",
			query:   "theatre",
			wantMin: 100,
		},
		{
			name:    "ID match",
			target:  "S01.11.15",
			query:   "11.15",
			wantMin: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := FuzzyScore(tt.target, tt.query)

			if tt.wantScore > 0 {
				if score != tt.wantScore {
					t.Errorf("expected score %d, got %d", tt.wantScore, score)
				}
			} else if tt.wantMin > 0 {
				if score < tt.wantMin {
					t.Errorf("expected score >= %d, got %d", tt.wantMin, score)
				}
			} else {
				if score != 0 {
					t.Errorf("expected score 0, got %d", score)
				}
			}
		})
	}
}

func TestFuzzyScore_Ordering(t *testing.T) {
	// Test that better matches score higher
	query := "theatre"

	exactScore := FuzzyScore("theatre", query)         // exact + prefix = 150
	prefixScore := FuzzyScore("theatre season", query) // contains + prefix = 150
	containsScore := FuzzyScore("my theatre", query)   // contains only = 100
	fuzzyScore := FuzzyScore("t.h.e.a.t.r.e", query)   // fuzzy match only

	if exactScore < prefixScore {
		t.Errorf("exact match should score >= prefix: %d < %d", exactScore, prefixScore)
	}
	if prefixScore < containsScore {
		t.Errorf("prefix match should score >= contains: %d < %d", prefixScore, containsScore)
	}
	if containsScore <= fuzzyScore {
		t.Errorf("contains match should score higher than fuzzy: %d <= %d", containsScore, fuzzyScore)
	}
}

func TestFuzzySort(t *testing.T) {
	results := []domain.SearchResult{
		{ID: "S01.11.99", Name: "Random Name", MatchedText: "nothing"},
		{ID: "S01.11.15", Name: "Theatre Season", MatchedText: "theatre"},
		{ID: "S01.12.11", Name: "Cooking", MatchedText: "recipes"},
		{ID: "S01.11.11", Name: "My Theatre", MatchedText: "old theatre"},
	}

	sorted := FuzzySort(results, "theatre")

	// Theatre Season should come first (prefix match in Name)
	if len(sorted) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(sorted))
	}

	// Verify that Theatre matches are ranked
	foundTheatre := false
	for _, r := range sorted {
		if r.Name == "Theatre Season" || r.Name == "My Theatre" {
			foundTheatre = true
		}
	}
	if !foundTheatre {
		t.Error("expected theatre matches in results")
	}

	// Verify results are sorted by score descending
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Score > sorted[i-1].Score {
			t.Errorf("results not sorted by score: %d > %d at index %d",
				sorted[i].Score, sorted[i-1].Score, i)
		}
	}
}
