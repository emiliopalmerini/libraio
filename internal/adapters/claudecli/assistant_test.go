package claudecli

import (
	"testing"

	"libraio/internal/ports"
)

func TestParseBatchSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		wantCount   int
		wantFirst   string // first file name
		wantFirstID string // first item ID
		wantErr     bool
	}{
		{
			name: "valid JSON array",
			result: `[
				{"fileName": "doc.pdf", "itemID": "S01.11.15", "itemName": "Theatre", "reasoning": "This file relates to theatre"},
				{"fileName": "notes.txt", "itemID": "S01.21.11", "itemName": "CSharp", "reasoning": "Programming notes"}
			]`,
			wantCount:   2,
			wantFirst:   "doc.pdf",
			wantFirstID: "S01.11.15",
			wantErr:     false,
		},
		{
			name:        "JSON in markdown code block",
			result:      "```json\n[{\"fileName\": \"file1.pdf\", \"itemID\": \"S01.21.11\", \"itemName\": \"Learning\", \"reasoning\": \"Educational content\"}]\n```",
			wantCount:   1,
			wantFirst:   "file1.pdf",
			wantFirstID: "S01.21.11",
			wantErr:     false,
		},
		{
			name:        "JSON with surrounding text",
			result:      "Here are my suggestions:\n[{\"fileName\": \"receipt.pdf\", \"itemID\": \"S02.15.12\", \"itemName\": \"Finance\", \"reasoning\": \"Financial document\"}]\nLet me know if you have questions.",
			wantCount:   1,
			wantFirst:   "receipt.pdf",
			wantFirstID: "S02.15.12",
			wantErr:     false,
		},
		{
			name:        "JSON in code block without language",
			result:      "```\n[{\"fileName\": \"test.txt\", \"itemID\": \"S01.11.11\", \"itemName\": \"Test Item\", \"reasoning\": \"Test reasoning\"}]\n```",
			wantCount:   1,
			wantFirst:   "test.txt",
			wantFirstID: "S01.11.11",
			wantErr:     false,
		},
		{
			name:        "missing fileName in one entry",
			result:      `[{"itemID": "S01.11.15", "itemName": "Test", "reasoning": "Test"}, {"fileName": "valid.pdf", "itemID": "S01.11.16", "itemName": "Valid", "reasoning": "Valid"}]`,
			wantCount:   1, // Only the valid entry
			wantFirst:   "valid.pdf",
			wantFirstID: "S01.11.16",
			wantErr:     false,
		},
		{
			name:    "no JSON array found",
			result:  "This is just plain text without any JSON",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			result:  `[{"fileName": "test.pdf", "itemID": }]`,
			wantErr: true,
		},
		{
			name:    "empty array",
			result:  `[]`,
			wantErr: true,
		},
		{
			name:    "all entries missing required fields",
			result:  `[{"reasoning": "Test"}, {"itemName": "Only name"}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := parseBatchSuggestions(tt.result)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseBatchSuggestions() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseBatchSuggestions() unexpected error: %v", err)
				return
			}

			if len(suggestions) != tt.wantCount {
				t.Errorf("got %d suggestions, want %d", len(suggestions), tt.wantCount)
				return
			}

			if tt.wantCount > 0 {
				if suggestions[0].FileName != tt.wantFirst {
					t.Errorf("first FileName = %q, want %q", suggestions[0].FileName, tt.wantFirst)
				}
				if suggestions[0].DestinationItemID != tt.wantFirstID {
					t.Errorf("first DestinationItemID = %q, want %q", suggestions[0].DestinationItemID, tt.wantFirstID)
				}
			}
		})
	}
}

func TestBuildBatchPrompt(t *testing.T) {
	files := []ports.FileInfo{
		{Name: "document.pdf", Content: "PDF content here"},
		{Name: "binary.exe", Content: ""},
	}

	vaultStructure := "S01.11.15 Theatre\nS01.21.11 Learning"
	prompt := buildBatchPrompt(files, vaultStructure)

	if !contains(prompt, "document.pdf") {
		t.Error("prompt should contain file name")
	}
	if !contains(prompt, "PDF content here") {
		t.Error("prompt should contain file content")
	}
	if !contains(prompt, "(Binary file") {
		t.Error("prompt should indicate binary file for empty content")
	}
	if !contains(prompt, vaultStructure) {
		t.Error("prompt should contain vault structure")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

func TestNewAssistant_DefaultModels(t *testing.T) {
	a := NewAssistant()
	if a.model != "haiku" {
		t.Errorf("default model = %q, want %q", a.model, "haiku")
	}
	if a.searchModel != "sonnet" {
		t.Errorf("default searchModel = %q, want %q", a.searchModel, "sonnet")
	}
}

func TestWithModel_DoesNotAffectSearchModel(t *testing.T) {
	a := NewAssistant(WithModel("opus"))
	if a.model != "opus" {
		t.Errorf("model = %q, want %q", a.model, "opus")
	}
	if a.searchModel != "sonnet" {
		t.Errorf("searchModel should remain %q, got %q", "sonnet", a.searchModel)
	}
}

func TestWithSearchModel(t *testing.T) {
	a := NewAssistant(WithSearchModel("opus"))
	if a.searchModel != "opus" {
		t.Errorf("searchModel = %q, want %q", a.searchModel, "opus")
	}
	if a.model != "haiku" {
		t.Errorf("model should remain %q, got %q", "haiku", a.model)
	}
}

func TestBuildSearchPrompt_ContainsKeyPhrases(t *testing.T) {
	prompt := buildSearchPrompt("movies", "S01.11 Entertainment")

	expectedPhrases := []string{
		"Johnny Decimal",
		`"movies"`,
		"S01.11 Entertainment",
		"semantic meaning",
		"synonyms",
		"Scopes",
		"Areas",
		"Categories",
		"Items",
		"JSON array",
	}

	for _, phrase := range expectedPhrases {
		if !contains(prompt, phrase) {
			t.Errorf("prompt missing expected phrase %q", phrase)
		}
	}
}

func TestParseSearchResults(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantCount int
		wantFirst string
		wantErr   bool
	}{
		{
			name:      "valid results",
			result:    `[{"path": "S01 Me/S01.10-19 Lifestyle/S01.11 Entertainment", "jdid": "S01.11", "name": "Entertainment", "type": "category", "score": 0.95, "reasoning": "test"}]`,
			wantCount: 1,
			wantFirst: "S01.11",
			wantErr:   false,
		},
		{
			name:      "empty array",
			result:    `[]`,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:    "no JSON",
			result:  "no results here",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := parseSearchResults(tt.result)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(results) != tt.wantCount {
				t.Errorf("got %d results, want %d", len(results), tt.wantCount)
				return
			}
			if tt.wantCount > 0 && results[0].JDID != tt.wantFirst {
				t.Errorf("first JDID = %q, want %q", results[0].JDID, tt.wantFirst)
			}
		})
	}
}
