package claudecli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"libraio/internal/ports"
)

// Assistant implements ports.AIAssistant using Claude Code CLI
type Assistant struct {
	model string
}

// Option configures the Assistant
type Option func(*Assistant)

// WithModel sets the Claude model to use
func WithModel(model string) Option {
	return func(a *Assistant) {
		a.model = model
	}
}

// NewAssistant creates a new Claude CLI assistant
func NewAssistant(opts ...Option) *Assistant {
	a := &Assistant{
		model: "haiku", // Default to haiku for speed
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// claudeResponse represents the JSON output from claude CLI
type claudeResponse struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	CostUSD      float64 `json:"cost_usd"`
	DurationMS   int     `json:"duration_ms"`
	DurationAPI  int     `json:"duration_api_ms"`
	IsError      bool    `json:"is_error"`
	NumTurns     int     `json:"num_turns"`
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// suggestionJSON represents the expected JSON format from Claude's response
type suggestionJSON struct {
	FileName  string `json:"fileName"`
	ItemID    string `json:"itemID"`
	ItemName  string `json:"itemName"`
	Reasoning string `json:"reasoning"`
	// Alternative suggestion
	AltItemID    string `json:"altItemID,omitempty"`
	AltItemName  string `json:"altItemName,omitempty"`
	AltReasoning string `json:"altReasoning,omitempty"`
}

// SuggestCatalogDestinations analyzes multiple files and suggests destinations
func (a *Assistant) SuggestCatalogDestinations(files []ports.FileInfo, vaultStructure string) ([]ports.CatalogSuggestion, error) {
	prompt := buildBatchPrompt(files, vaultStructure)

	// Call claude CLI with JSON output
	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--model", a.model,
	}

	cmd := exec.Command("claude", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude CLI error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("claude CLI error: %w", err)
	}

	// Parse the claude CLI JSON response
	var response claudeResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse claude response: %w", err)
	}

	if response.IsError {
		return nil, fmt.Errorf("claude returned an error: %s", response.Result)
	}

	// Parse the suggestions from Claude's response text
	return parseBatchSuggestions(response.Result)
}

func buildBatchPrompt(files []ports.FileInfo, vaultStructure string) string {
	var filesList strings.Builder
	for i, f := range files {
		filesList.WriteString(fmt.Sprintf("\n### File %d: %s\n", i+1, f.Name))
		if f.Content != "" {
			filesList.WriteString(fmt.Sprintf("Content:\n%s\n", f.Content))
		} else {
			filesList.WriteString("(Binary file - no content preview)\n")
		}
	}

	return fmt.Sprintf(`You are helping organize files in a Johnny Decimal vault.

Analyze these files from the inbox and suggest where each should be moved:
%s

Available items in this vault:
%s

For EACH file, suggest TWO destinations ranked by relevance:
1. Primary: the best existing item
2. Alternative: a second-best option (different from primary)

Return ONLY a JSON array (no markdown, no code blocks):
[
  {"fileName": "file1.pdf", "itemID": "S01.11.15", "itemName": "Theatre", "reasoning": "Brief explanation", "altItemID": "S01.11.16", "altItemName": "Movies", "altReasoning": "Alternative explanation"},
  {"fileName": "file2.txt", "itemID": "S01.21.11", "itemName": "CSharp", "reasoning": "Brief explanation", "altItemID": "S01.21.12", "altItemName": "Programming", "altReasoning": "Alternative explanation"}
]`, filesList.String(), vaultStructure)
}

// parseBatchSuggestions extracts the suggestions JSON array from Claude's response
func parseBatchSuggestions(result string) ([]ports.CatalogSuggestion, error) {
	result = strings.TrimSpace(result)

	// Try to extract JSON from markdown code blocks if present
	codeBlockRe := regexp.MustCompile("```(?:json)?\\s*\\n?([\\s\\S]*?)\\n?```")
	if matches := codeBlockRe.FindStringSubmatch(result); len(matches) > 1 {
		result = strings.TrimSpace(matches[1])
	}

	// Find JSON array in the text (handles surrounding text)
	jsonStartIdx := strings.Index(result, "[")
	jsonEndIdx := strings.LastIndex(result, "]")
	if jsonStartIdx == -1 || jsonEndIdx == -1 || jsonEndIdx <= jsonStartIdx {
		return nil, fmt.Errorf("no valid JSON array found in response")
	}

	jsonStr := result[jsonStartIdx : jsonEndIdx+1]

	var rawSuggestions []suggestionJSON
	if err := json.Unmarshal([]byte(jsonStr), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("failed to parse suggestions JSON: %w (json: %s)", err, jsonStr)
	}

	// Convert to ports.CatalogSuggestion, validate each has required fields
	var suggestions []ports.CatalogSuggestion
	for _, raw := range rawSuggestions {
		if raw.FileName == "" || raw.ItemID == "" {
			continue // Skip invalid entries
		}
		suggestions = append(suggestions, ports.CatalogSuggestion{
			FileName:            raw.FileName,
			DestinationItemID:   raw.ItemID,
			DestinationItemName: raw.ItemName,
			Reasoning:           raw.Reasoning,
			// Alternative suggestion
			AltDestinationItemID:   raw.AltItemID,
			AltDestinationItemName: raw.AltItemName,
			AltReasoning:           raw.AltReasoning,
		})
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no valid suggestions found in response")
	}

	return suggestions, nil
}
