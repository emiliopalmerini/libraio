package commands

import (
	"testing"

	"libraio/internal/domain"
)

func TestMoveItemCommand_Validate(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		destID   string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid item to category",
			sourceID: "S01.11.15",
			destID:   "S01.12",
			wantErr:  false,
		},
		{
			name:     "empty source ID",
			sourceID: "",
			destID:   "S01.12",
			wantErr:  true,
			errMsg:   "source item ID is required",
		},
		{
			name:     "empty destination ID",
			sourceID: "S01.11.15",
			destID:   "",
			wantErr:  true,
			errMsg:   "destination category ID is required",
		},
		{
			name:     "source is not item",
			sourceID: "S01.11",
			destID:   "S01.12",
			wantErr:  true,
			errMsg:   "source must be an item",
		},
		{
			name:     "destination is not category",
			sourceID: "S01.11.15",
			destID:   "S01.10-19",
			wantErr:  true,
			errMsg:   "items can only be moved to categories",
		},
		{
			name:     "destination is item",
			sourceID: "S01.11.15",
			destID:   "S01.12.11",
			wantErr:  true,
			errMsg:   "items can only be moved to categories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &MoveItemCommand{
				SourceItemID:     tt.sourceID,
				DestinationCatID: tt.destID,
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

func TestMoveCategoryCommand_Validate(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		destID   string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid category to area",
			sourceID: "S01.11",
			destID:   "S01.20-29",
			wantErr:  false,
		},
		{
			name:     "empty source ID",
			sourceID: "",
			destID:   "S01.20-29",
			wantErr:  true,
			errMsg:   "source category ID is required",
		},
		{
			name:     "empty destination ID",
			sourceID: "S01.11",
			destID:   "",
			wantErr:  true,
			errMsg:   "destination area ID is required",
		},
		{
			name:     "source is not category",
			sourceID: "S01.11.15",
			destID:   "S01.20-29",
			wantErr:  true,
			errMsg:   "source must be a category",
		},
		{
			name:     "destination is not area",
			sourceID: "S01.11",
			destID:   "S01.21",
			wantErr:  true,
			errMsg:   "categories can only be moved to areas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &MoveCategoryCommand{
				SourceCatID:     tt.sourceID,
				DestinationArea: tt.destID,
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

func TestValidateMoveDestination(t *testing.T) {
	tests := []struct {
		name       string
		sourceID   string
		sourceType domain.IDType
		destID     string
		wantErr    bool
	}{
		{
			name:       "item to category - valid",
			sourceID:   "S01.11.15",
			sourceType: domain.IDTypeItem,
			destID:     "S01.12",
			wantErr:    false,
		},
		{
			name:       "item to area - invalid",
			sourceID:   "S01.11.15",
			sourceType: domain.IDTypeItem,
			destID:     "S01.20-29",
			wantErr:    true,
		},
		{
			name:       "category to area - valid",
			sourceID:   "S01.11",
			sourceType: domain.IDTypeCategory,
			destID:     "S01.20-29",
			wantErr:    false,
		},
		{
			name:       "category to category - invalid",
			sourceID:   "S01.11",
			sourceType: domain.IDTypeCategory,
			destID:     "S01.21",
			wantErr:    true,
		},
		{
			name:       "area cannot be moved",
			sourceID:   "S01.10-19",
			sourceType: domain.IDTypeArea,
			destID:     "S02.10-19",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMoveDestination(tt.sourceID, tt.sourceType, tt.destID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
