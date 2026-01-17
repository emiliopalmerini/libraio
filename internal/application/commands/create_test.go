package commands

import (
	"testing"

	"libraio/internal/domain"
)

func TestCreateItemCommand_Validate(t *testing.T) {
	tests := []struct {
		name        string
		categoryID  string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid create item",
			categoryID:  "S01.11",
			description: "New Item",
			wantErr:     false,
		},
		{
			name:        "empty category ID",
			categoryID:  "",
			description: "New Item",
			wantErr:     true,
			errMsg:      "category ID is required",
		},
		{
			name:        "empty description",
			categoryID:  "S01.11",
			description: "",
			wantErr:     true,
			errMsg:      "description is required",
		},
		{
			name:        "invalid parent type - area",
			categoryID:  "S01.10-19",
			description: "New Item",
			wantErr:     true,
			errMsg:      "expected category ID",
		},
		{
			name:        "invalid parent type - item",
			categoryID:  "S01.11.15",
			description: "New Item",
			wantErr:     true,
			errMsg:      "expected category ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CreateItemCommand{
				CategoryID:  tt.categoryID,
				Description: tt.description,
			}
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

func TestCreateCategoryCommand_Validate(t *testing.T) {
	tests := []struct {
		name        string
		areaID      string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid create category",
			areaID:      "S01.10-19",
			description: "New Category",
			wantErr:     false,
		},
		{
			name:        "empty area ID",
			areaID:      "",
			description: "New Category",
			wantErr:     true,
			errMsg:      "area ID is required",
		},
		{
			name:        "empty description",
			areaID:      "S01.10-19",
			description: "",
			wantErr:     true,
			errMsg:      "description is required",
		},
		{
			name:        "invalid parent type - category",
			areaID:      "S01.11",
			description: "New Category",
			wantErr:     true,
			errMsg:      "expected area ID",
		},
		{
			name:        "invalid parent type - scope",
			areaID:      "S01",
			description: "New Category",
			wantErr:     true,
			errMsg:      "expected area ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CreateCategoryCommand{
				AreaID:      tt.areaID,
				Description: tt.description,
			}
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

func TestDetermineCreateMode(t *testing.T) {
	tests := []struct {
		name       string
		parentType domain.IDType
		wantMode   CreateMode
		wantErr    bool
	}{
		{
			name:       "area creates category",
			parentType: domain.IDTypeArea,
			wantMode:   CreateModeCategory,
			wantErr:    false,
		},
		{
			name:       "category creates item",
			parentType: domain.IDTypeCategory,
			wantMode:   CreateModeItem,
			wantErr:    false,
		},
		{
			name:       "scope cannot create",
			parentType: domain.IDTypeScope,
			wantErr:    true,
		},
		{
			name:       "item cannot create",
			parentType: domain.IDTypeItem,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := DetermineCreateMode(tt.parentType)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if mode != tt.wantMode {
					t.Errorf("expected mode %v, got %v", tt.wantMode, mode)
				}
			}
		})
	}
}
