package commands

import (
	"testing"

	"libraio/internal/domain"
)

func TestArchiveItemCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		itemID  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid item ID",
			itemID:  "S01.11.15",
			wantErr: false,
		},
		{
			name:    "empty item ID",
			itemID:  "",
			wantErr: true,
			errMsg:  "item ID is required",
		},
		{
			name:    "invalid ID type - category",
			itemID:  "S01.11",
			wantErr: true,
			errMsg:  "expected item ID",
		},
		{
			name:    "item already in archive category",
			itemID:  "S01.19.15",
			wantErr: true,
			errMsg:  "already in an archive category",
		},
		{
			name:    "item in archive category S02",
			itemID:  "S02.29.11",
			wantErr: true,
			errMsg:  "already in an archive category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ArchiveItemCommand{ItemID: tt.itemID}
			err := cmd.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestArchiveCategoryCommand_Validate(t *testing.T) {
	tests := []struct {
		name       string
		categoryID string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid category ID",
			categoryID: "S01.11",
			wantErr:    false,
		},
		{
			name:       "empty category ID",
			categoryID: "",
			wantErr:    true,
			errMsg:     "category ID is required",
		},
		{
			name:       "invalid ID type - item",
			categoryID: "S01.11.15",
			wantErr:    true,
			errMsg:     "expected category ID",
		},
		{
			name:       "archive category itself",
			categoryID: "S01.19",
			wantErr:    true,
			errMsg:     "cannot archive the archive category",
		},
		{
			name:       "archive category S02",
			categoryID: "S02.29",
			wantErr:    true,
			errMsg:     "cannot archive the archive category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ArchiveCategoryCommand{CategoryID: tt.categoryID}
			err := cmd.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckArchiveEligibility(t *testing.T) {
	tests := []struct {
		name       string
		nodeID     string
		nodeType   domain.IDType
		canArchive bool
	}{
		{
			name:       "item can be archived",
			nodeID:     "S01.11.15",
			nodeType:   domain.IDTypeItem,
			canArchive: true,
		},
		{
			name:       "category can be archived",
			nodeID:     "S01.11",
			nodeType:   domain.IDTypeCategory,
			canArchive: true,
		},
		{
			name:       "item in archive cannot be archived",
			nodeID:     "S01.19.15",
			nodeType:   domain.IDTypeItem,
			canArchive: false,
		},
		{
			name:       "archive category cannot be archived",
			nodeID:     "S01.19",
			nodeType:   domain.IDTypeCategory,
			canArchive: false,
		},
		{
			name:       "area cannot be archived",
			nodeID:     "S01.10-19",
			nodeType:   domain.IDTypeArea,
			canArchive: false,
		},
		{
			name:       "scope cannot be archived",
			nodeID:     "S01",
			nodeType:   domain.IDTypeScope,
			canArchive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckArchiveEligibility(tt.nodeID, tt.nodeType)

			if result.CanArchive != tt.canArchive {
				t.Errorf("expected CanArchive=%v, got %v (reason: %s)", tt.canArchive, result.CanArchive, result.Reason)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
