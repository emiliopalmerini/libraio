package commands

import (
	"context"
	"sort"
	"strings"

	"libraio/internal/domain"
	"libraio/internal/ports"
)

// SearchResult wraps domain.SearchResult with a relevance score
type SearchResult struct {
	domain.SearchResult
	Score int
}

// SearchCommand searches the vault with fuzzy matching
type SearchCommand struct {
	repo  ports.VaultRepository
	Query string
}

// NewSearchCommand creates a new SearchCommand
func NewSearchCommand(repo ports.VaultRepository, query string) *SearchCommand {
	return &SearchCommand{
		repo:  repo,
		Query: query,
	}
}

// Execute runs the search command and returns scored, sorted results
func (c *SearchCommand) Execute(ctx context.Context) ([]SearchResult, error) {
	if len(c.Query) < 2 {
		return nil, nil
	}

	results, err := c.repo.Search(c.Query)
	if err != nil {
		return nil, err
	}

	return FuzzySort(results, c.Query), nil
}

// FuzzyScore calculates a relevance score for how well target matches query
func FuzzyScore(target, query string) int {
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
				score += 10 // consecutive chars
			}
			if i == 0 {
				score += 15 // start of string
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

// FuzzySort sorts search results by relevance to the query
func FuzzySort(results []domain.SearchResult, query string) []SearchResult {
	scored := make([]SearchResult, 0, len(results))

	for _, r := range results {
		s1 := FuzzyScore(r.ID, query)
		s2 := FuzzyScore(r.Name, query)
		s3 := FuzzyScore(r.MatchedText, query)

		best := max(s1, s2, s3)

		if best > 0 {
			scored = append(scored, SearchResult{
				SearchResult: r,
				Score:        best,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored
}
