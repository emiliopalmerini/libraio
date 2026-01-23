package views

import (
	"strings"

	"libraio/internal/application"
)

// SearchScorer provides fuzzy search scoring for search results
type SearchScorer struct{}

// NewSearchScorer creates a new search scorer
func NewSearchScorer() *SearchScorer {
	return &SearchScorer{}
}

// FuzzyScore calculates a score for how well target matches query.
// Higher scores indicate better matches.
// Returns 0 if there is no match.
func (s *SearchScorer) FuzzyScore(target, query string) int {
	target = strings.ToLower(target)
	query = strings.ToLower(query)

	if len(query) == 0 {
		return 0
	}

	// Check for exact substring match first (highest priority)
	if strings.Contains(target, query) {
		score := 100
		// Bonus if it starts with query
		if strings.HasPrefix(target, query) {
			score += 50
		}
		return score
	}

	// Fuzzy match: check if chars appear in order
	score := 0
	queryIdx := 0
	prevMatchIdx := -1

	for i := 0; i < len(target) && queryIdx < len(query); i++ {
		if target[i] == query[queryIdx] {
			if prevMatchIdx == i-1 {
				score += 10 // consecutive
			}
			if i == 0 {
				score += 15 // start
			}
			if i > 0 && (target[i-1] == ' ' || target[i-1] == '.' || target[i-1] == '-') {
				score += 10 // after separator
			}
			score += 1
			prevMatchIdx = i
			queryIdx++
		}
	}

	if queryIdx == len(query) {
		return score
	}
	return 0
}

// SortResults sorts search results by relevance to query.
// Results with no match are filtered out.
func (s *SearchScorer) SortResults(results []application.SearchResult, query string) []application.SearchResult {
	type scored struct {
		result application.SearchResult
		score  int
	}

	var scoredResults []scored
	for _, r := range results {
		s1 := s.FuzzyScore(r.ID, query)
		s2 := s.FuzzyScore(r.Name, query)
		s3 := s.FuzzyScore(r.MatchedText, query)
		best := max(s3, max(s2, s1))
		if best > 0 {
			scoredResults = append(scoredResults, scored{result: r, score: best})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scoredResults)-1; i++ {
		for j := i + 1; j < len(scoredResults); j++ {
			if scoredResults[j].score > scoredResults[i].score {
				scoredResults[i], scoredResults[j] = scoredResults[j], scoredResults[i]
			}
		}
	}

	sorted := make([]application.SearchResult, len(scoredResults))
	for i, s := range scoredResults {
		sorted[i] = s.result
	}
	return sorted
}
