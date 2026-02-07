package commands

import (
	"testing"

	"libraio/internal/domain"
)

func TestUnarchiveItemCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		itemID  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid archive item",
			itemID:  "S01.11.09",
			wantErr: false,
		},
		{
			name:    "empty ID",
			itemID:  "",
			wantErr: true,
			errMsg:  "item ID is required",
		},
		{
			name:    "not an item",
			itemID:  "S01.11",
			wantErr: true,
			errMsg:  "expected item ID",
		},
		{
			name:    "not an archive item",
			itemID:  "S01.11.15",
			wantErr: true,
			errMsg:  "not an archive item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &UnarchiveItemCommand{
				ArchiveItemID: tt.itemID,
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

func TestUnarchiveEligibility(t *testing.T) {
	tests := []struct {
		name         string
		nodeID       string
		nodeType     domain.IDType
		canUnarchive bool
	}{
		{name: "archive item can be unarchived", nodeID: "S01.11.09", nodeType: domain.IDTypeItem, canUnarchive: true},
		{name: "regular item cannot be unarchived", nodeID: "S01.11.15", nodeType: domain.IDTypeItem, canUnarchive: false},
		{name: "category cannot be unarchived", nodeID: "S01.11", nodeType: domain.IDTypeCategory, canUnarchive: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckUnarchiveEligibility(tt.nodeID, tt.nodeType)
			if result.CanUnarchive != tt.canUnarchive {
				t.Errorf("expected canUnarchive=%v, got %v (reason: %s)", tt.canUnarchive, result.CanUnarchive, result.Reason)
			}
		})
	}
}
