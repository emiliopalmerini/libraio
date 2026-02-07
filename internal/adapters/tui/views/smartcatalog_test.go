package views

import (
	"testing"

	"libraio/internal/application"
	"libraio/internal/domain"
	"libraio/internal/ports"
)

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: true,
		},
		{
			name:     "plain text",
			content:  []byte("Hello, World!"),
			expected: true,
		},
		{
			name:     "text with newlines",
			content:  []byte("Line 1\nLine 2\nLine 3"),
			expected: true,
		},
		{
			name:     "markdown content",
			content:  []byte("# Title\n\nSome **bold** text"),
			expected: true,
		},
		{
			name:     "binary with null byte at start",
			content:  []byte{0x00, 0x01, 0x02},
			expected: false,
		},
		{
			name:     "binary with null byte in middle",
			content:  []byte("hello\x00world"),
			expected: false,
		},
		{
			name:     "text longer than 512 bytes without nulls",
			content:  make([]byte, 600), // Will be filled with zeros, but let's fix that
			expected: false,             // zeros are null bytes
		},
	}

	// Fix the last test case - create actual text content
	longText := make([]byte, 600)
	for i := range longText {
		longText[i] = 'a'
	}
	tests[len(tests)-1] = struct {
		name     string
		content  []byte
		expected bool
	}{
		name:     "text longer than 512 bytes without nulls",
		content:  longText,
		expected: true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextContent(tt.content)
			if result != tt.expected {
				t.Errorf("isTextContent() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMatchSuggestionsToFiles(t *testing.T) {
	tests := []struct {
		name        string
		files       []ports.FileInfo
		suggestions []ports.CatalogSuggestion
		wantCount   int
		wantMatched int // number of files that should have suggestions
	}{
		{
			name:        "empty files and suggestions",
			files:       nil,
			suggestions: nil,
			wantCount:   0,
			wantMatched: 0,
		},
		{
			name: "files without suggestions",
			files: []ports.FileInfo{
				{Name: "file1.txt", Path: "/inbox/file1.txt"},
				{Name: "file2.txt", Path: "/inbox/file2.txt"},
			},
			suggestions: nil,
			wantCount:   2,
			wantMatched: 0,
		},
		{
			name: "all files have suggestions",
			files: []ports.FileInfo{
				{Name: "file1.txt", Path: "/inbox/file1.txt"},
				{Name: "file2.txt", Path: "/inbox/file2.txt"},
			},
			suggestions: []ports.CatalogSuggestion{
				{FileName: "file1.txt", DestinationItemID: "S01.11.15", DestinationItemName: "Project A"},
				{FileName: "file2.txt", DestinationItemID: "S01.12.11", DestinationItemName: "Project B"},
			},
			wantCount:   2,
			wantMatched: 2,
		},
		{
			name: "partial match",
			files: []ports.FileInfo{
				{Name: "file1.txt", Path: "/inbox/file1.txt"},
				{Name: "file2.txt", Path: "/inbox/file2.txt"},
				{Name: "file3.txt", Path: "/inbox/file3.txt"},
			},
			suggestions: []ports.CatalogSuggestion{
				{FileName: "file1.txt", DestinationItemID: "S01.11.15", DestinationItemName: "Project A"},
			},
			wantCount:   3,
			wantMatched: 1,
		},
		{
			name: "suggestion for non-existent file is ignored",
			files: []ports.FileInfo{
				{Name: "file1.txt", Path: "/inbox/file1.txt"},
			},
			suggestions: []ports.CatalogSuggestion{
				{FileName: "file1.txt", DestinationItemID: "S01.11.15", DestinationItemName: "Project A"},
				{FileName: "nonexistent.txt", DestinationItemID: "S01.12.11", DestinationItemName: "Project B"},
			},
			wantCount:   1,
			wantMatched: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchSuggestionsToFiles(tt.files, tt.suggestions)

			if len(result) != tt.wantCount {
				t.Errorf("matchSuggestionsToFiles() returned %d items, expected %d", len(result), tt.wantCount)
			}

			matched := 0
			for _, fs := range result {
				if fs.Suggestion != nil {
					matched++
				}
			}
			if matched != tt.wantMatched {
				t.Errorf("matchSuggestionsToFiles() matched %d files, expected %d", matched, tt.wantMatched)
			}
		})
	}
}

func TestSwapToAlternatives(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []FileSuggestion
		wantPrimary []string // expected primary destination IDs after swap
	}{
		{
			name:        "empty suggestions",
			suggestions: nil,
			wantPrimary: nil,
		},
		{
			name: "suggestion with alternative - should swap",
			suggestions: []FileSuggestion{
				{
					File: ports.FileInfo{Name: "file1.txt"},
					Suggestion: &ports.CatalogSuggestion{
						FileName:               "file1.txt",
						DestinationItemID:      "S01.11.15",
						DestinationItemName:    "Primary",
						Reasoning:              "Primary reason",
						AltDestinationItemID:   "S01.12.11",
						AltDestinationItemName: "Alternative",
						AltReasoning:           "Alt reason",
					},
				},
			},
			wantPrimary: []string{"S01.12.11"},
		},
		{
			name: "suggestion without alternative - keeps original",
			suggestions: []FileSuggestion{
				{
					File: ports.FileInfo{Name: "file1.txt"},
					Suggestion: &ports.CatalogSuggestion{
						FileName:            "file1.txt",
						DestinationItemID:   "S01.11.15",
						DestinationItemName: "Primary",
						Reasoning:           "Primary reason",
					},
				},
			},
			wantPrimary: []string{"S01.11.15"},
		},
		{
			name: "nil suggestion - stays nil",
			suggestions: []FileSuggestion{
				{
					File:       ports.FileInfo{Name: "file1.txt"},
					Suggestion: nil,
				},
			},
			wantPrimary: []string{""},
		},
		{
			name: "mixed suggestions",
			suggestions: []FileSuggestion{
				{
					File: ports.FileInfo{Name: "file1.txt"},
					Suggestion: &ports.CatalogSuggestion{
						FileName:               "file1.txt",
						DestinationItemID:      "S01.11.15",
						AltDestinationItemID:   "S01.12.11",
						AltDestinationItemName: "Alt1",
					},
				},
				{
					File: ports.FileInfo{Name: "file2.txt"},
					Suggestion: &ports.CatalogSuggestion{
						FileName:          "file2.txt",
						DestinationItemID: "S01.13.11",
						// No alternative
					},
				},
				{
					File:       ports.FileInfo{Name: "file3.txt"},
					Suggestion: nil,
				},
			},
			wantPrimary: []string{"S01.12.11", "S01.13.11", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := swapToAlternatives(tt.suggestions)

			if len(result) != len(tt.wantPrimary) {
				t.Errorf("swapToAlternatives() returned %d items, expected %d", len(result), len(tt.wantPrimary))
				return
			}

			for i, fs := range result {
				var gotID string
				if fs.Suggestion != nil {
					gotID = fs.Suggestion.DestinationItemID
				}
				if gotID != tt.wantPrimary[i] {
					t.Errorf("swapToAlternatives()[%d].DestinationItemID = %q, expected %q", i, gotID, tt.wantPrimary[i])
				}
			}
		})
	}
}

func TestSwapToAlternatives_PreservesOriginalAsAlt(t *testing.T) {
	suggestions := []FileSuggestion{
		{
			File: ports.FileInfo{Name: "file1.txt"},
			Suggestion: &ports.CatalogSuggestion{
				FileName:               "file1.txt",
				DestinationItemID:      "S01.11.15",
				DestinationItemName:    "Primary",
				Reasoning:              "Primary reason",
				AltDestinationItemID:   "S01.12.11",
				AltDestinationItemName: "Alternative",
				AltReasoning:           "Alt reason",
			},
		},
	}

	result := swapToAlternatives(suggestions)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	s := result[0].Suggestion
	// Primary should now be the alternative
	if s.DestinationItemID != "S01.12.11" {
		t.Errorf("DestinationItemID = %q, expected S01.12.11", s.DestinationItemID)
	}
	if s.DestinationItemName != "Alternative" {
		t.Errorf("DestinationItemName = %q, expected Alternative", s.DestinationItemName)
	}
	if s.Reasoning != "Alt reason" {
		t.Errorf("Reasoning = %q, expected Alt reason", s.Reasoning)
	}

	// Alternative should now be the original primary
	if s.AltDestinationItemID != "S01.11.15" {
		t.Errorf("AltDestinationItemID = %q, expected S01.11.15", s.AltDestinationItemID)
	}
	if s.AltDestinationItemName != "Primary" {
		t.Errorf("AltDestinationItemName = %q, expected Primary", s.AltDestinationItemName)
	}
	if s.AltReasoning != "Primary reason" {
		t.Errorf("AltReasoning = %q, expected Primary reason", s.AltReasoning)
	}
}

// Mock implementations for testing

type mockVaultRepository struct {
	pathMap    map[string]string
	items      map[string][]domain.Item
	categories map[string][]domain.Category
	areas      map[string][]domain.Area
	scopes     []domain.Scope
}

func newMockVaultRepository() *mockVaultRepository {
	return &mockVaultRepository{
		pathMap:    make(map[string]string),
		items:      make(map[string][]domain.Item),
		categories: make(map[string][]domain.Category),
		areas:      make(map[string][]domain.Area),
	}
}

func (m *mockVaultRepository) GetPath(id string) (string, error) {
	if path, ok := m.pathMap[id]; ok {
		return path, nil
	}
	return "", nil
}

func (m *mockVaultRepository) GetJDexPath(itemID string) (string, error) {
	return "", nil
}

func (m *mockVaultRepository) ListItems(categoryID string) ([]domain.Item, error) {
	return m.items[categoryID], nil
}

func (m *mockVaultRepository) ListCategories(areaID string) ([]domain.Category, error) {
	return m.categories[areaID], nil
}

func (m *mockVaultRepository) ListAreas(scopeID string) ([]domain.Area, error) {
	return m.areas[scopeID], nil
}

func (m *mockVaultRepository) ListScopes() ([]domain.Scope, error) {
	return m.scopes, nil
}

func (m *mockVaultRepository) BuildTree() (*domain.TreeNode, error) { return nil, nil }
func (m *mockVaultRepository) LoadChildren(*domain.TreeNode) error  { return nil }
func (m *mockVaultRepository) Search(string) ([]domain.SearchResult, error) {
	return nil, nil
}
func (m *mockVaultRepository) CreateScope(string) (*domain.Scope, error)       { return nil, nil }
func (m *mockVaultRepository) CreateArea(string, string) (*domain.Area, error) { return nil, nil }
func (m *mockVaultRepository) CreateCategory(string, string) (*domain.Category, error) {
	return nil, nil
}
func (m *mockVaultRepository) CreateItem(string, string) (*domain.Item, error) { return nil, nil }
func (m *mockVaultRepository) MoveItem(string, string) (*domain.Item, error)   { return nil, nil }
func (m *mockVaultRepository) MoveCategory(string, string) (*domain.Category, error) {
	return nil, nil
}
func (m *mockVaultRepository) ArchiveItem(string) (*domain.Item, error)       { return nil, nil }
func (m *mockVaultRepository) ArchiveCategory(string) ([]*domain.Item, error) { return nil, nil }
func (m *mockVaultRepository) UnarchiveItems(string, string) ([]*domain.Item, error) {
	return nil, nil
}
func (m *mockVaultRepository) RenameItem(string, string) (*domain.Item, error) { return nil, nil }
func (m *mockVaultRepository) RenameCategory(string, string) (*domain.Category, error) {
	return nil, nil
}
func (m *mockVaultRepository) RenameArea(string, string) (*domain.Area, error) { return nil, nil }
func (m *mockVaultRepository) Delete(string) error                             { return nil }
func (m *mockVaultRepository) VaultPath() string                               { return "/mock/vault" }

type mockAIAssistant struct {
	suggestions []ports.CatalogSuggestion
	err         error
}

func (m *mockAIAssistant) SuggestCatalogDestinations(files []ports.FileInfo, vaultStructure string) ([]ports.CatalogSuggestion, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.suggestions, nil
}

func TestBuildVaultContextForLevel_Category(t *testing.T) {
	repo := newMockVaultRepository()
	repo.items["S01.11"] = []domain.Item{
		{ID: "S01.11.01", Name: "Inbox for S01.11"},
		{ID: "S01.11.09", Name: "Archive for S01.11"},
		{ID: "S01.11.11", Name: "Project Alpha"},
		{ID: "S01.11.15", Name: "Project Beta"},
	}

	result, err := buildVaultContextForLevel(repo, "S01.11", domain.InboxLevelCategory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only include items with num > 9 (standard zeros excluded)
	if !contains(result, "S01.11.11") {
		t.Errorf("expected S01.11.11 in result, got: %s", result)
	}
	if !contains(result, "S01.11.15") {
		t.Errorf("expected S01.11.15 in result, got: %s", result)
	}
	// Standard zeros should be excluded
	if contains(result, "S01.11.01") {
		t.Errorf("expected S01.11.01 (inbox) to be excluded, got: %s", result)
	}
	if contains(result, "S01.11.09") {
		t.Errorf("expected S01.11.09 (archive) to be excluded, got: %s", result)
	}
}

func TestBuildVaultContextForLevel_Area(t *testing.T) {
	repo := newMockVaultRepository()
	repo.categories["S01.10-19"] = []domain.Category{
		{ID: "S01.10", Name: "Management for S01.10-19"},
		{ID: "S01.11", Name: "Entertainment"},
		{ID: "S01.12", Name: "Health"},
	}
	repo.items["S01.11"] = []domain.Item{
		{ID: "S01.11.11", Name: "Movies"},
		{ID: "S01.11.12", Name: "Games"},
	}
	repo.items["S01.12"] = []domain.Item{
		{ID: "S01.12.11", Name: "Exercise"},
	}

	result, err := buildVaultContextForLevel(repo, "S01.10-19", domain.InboxLevelArea)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include categories and their items (excluding management category)
	if !contains(result, "S01.11 Entertainment") {
		t.Errorf("expected S01.11 Entertainment in result, got: %s", result)
	}
	if !contains(result, "S01.11.11 Movies") {
		t.Errorf("expected S01.11.11 Movies in result, got: %s", result)
	}
	// Management category should be excluded
	if contains(result, "S01.10 Management") {
		t.Errorf("expected S01.10 Management to be excluded, got: %s", result)
	}
}

func TestSmartCatalogModel_HandleFileMoved(t *testing.T) {
	model := NewSmartCatalogModel(nil, nil)
	model.suggestions = []FileSuggestion{
		{File: ports.FileInfo{Name: "file1.txt"}},
		{File: ports.FileInfo{Name: "file2.txt"}},
		{File: ports.FileInfo{Name: "file3.txt"}},
	}
	model.paginator.SetTotal(3)

	// Move first file
	model.HandleFileMoved()

	if model.moved != 1 {
		t.Errorf("moved count = %d, expected 1", model.moved)
	}
	if len(model.suggestions) != 2 {
		t.Errorf("suggestions count = %d, expected 2", len(model.suggestions))
	}
}

func TestSmartCatalogModel_IsEmpty(t *testing.T) {
	model := NewSmartCatalogModel(nil, nil)

	if !model.IsEmpty() {
		t.Error("expected IsEmpty() to return true for new model")
	}

	model.suggestions = []FileSuggestion{
		{File: ports.FileInfo{Name: "file1.txt"}},
	}

	if model.IsEmpty() {
		t.Error("expected IsEmpty() to return false when suggestions exist")
	}
}

func TestSmartCatalogModel_SetSource(t *testing.T) {
	model := NewSmartCatalogModel(nil, nil)
	model.moved = 5
	model.suggestions = []FileSuggestion{
		{File: ports.FileInfo{Name: "file1.txt"}},
	}
	model.skipped = []FileSuggestion{
		{File: ports.FileInfo{Name: "file2.txt"}},
	}
	model.reviewMode = true

	node := &application.TreeNode{
		ID:   "S01.11.01",
		Name: "Inbox for S01.11",
	}

	model.SetSource(node)

	if model.inboxNode != node {
		t.Error("expected inboxNode to be set")
	}
	if model.moved != 0 {
		t.Errorf("expected moved to be reset to 0, got %d", model.moved)
	}
	if len(model.suggestions) != 0 {
		t.Errorf("expected suggestions to be cleared, got %d", len(model.suggestions))
	}
	if len(model.skipped) != 0 {
		t.Errorf("expected skipped to be cleared, got %d", len(model.skipped))
	}
	if model.reviewMode {
		t.Error("expected reviewMode to be reset to false")
	}
	if model.state != SmartCatalogLoading {
		t.Errorf("expected state to be SmartCatalogLoading, got %v", model.state)
	}
}

func TestSmartCatalogModel_VisibleSuggestions(t *testing.T) {
	model := NewSmartCatalogModel(nil, nil)

	// Empty suggestions
	visible := model.visibleSuggestions()
	if visible != nil {
		t.Errorf("expected nil for empty suggestions, got %v", visible)
	}

	// Add some suggestions
	model.suggestions = make([]FileSuggestion, 25)
	for i := range 25 {
		model.suggestions[i] = FileSuggestion{
			File: ports.FileInfo{Name: "file" + string(rune('a'+i)) + ".txt"},
		}
	}
	model.paginator.SetTotal(25)

	// First page should have 10 items (default page size)
	visible = model.visibleSuggestions()
	if len(visible) != 10 {
		t.Errorf("expected 10 visible items on first page, got %d", len(visible))
	}

	// Move to next page
	model.paginator.NextPage()
	visible = model.visibleSuggestions()
	if len(visible) != 10 {
		t.Errorf("expected 10 visible items on second page, got %d", len(visible))
	}

	// Move to last page
	model.paginator.NextPage()
	visible = model.visibleSuggestions()
	if len(visible) != 5 {
		t.Errorf("expected 5 visible items on last page, got %d", len(visible))
	}
}

// Helper function for tests
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
