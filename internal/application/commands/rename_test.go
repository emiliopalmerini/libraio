package commands

import (
	"testing"

	"libraio/internal/domain"
)

func TestRenameCommand_Validate(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		newDescription string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "valid item rename",
			id:             "S01.11.15",
			newDescription: "New Name",
			wantErr:        false,
		},
		{
			name:           "valid category rename",
			id:             "S01.11",
			newDescription: "New Category",
			wantErr:        false,
		},
		{
			name:           "valid area rename",
			id:             "S01.10-19",
			newDescription: "New Area",
			wantErr:        false,
		},
		{
			name:           "empty ID",
			id:             "",
			newDescription: "Name",
			wantErr:        true,
			errMsg:         "ID is required",
		},
		{
			name:           "empty description",
			id:             "S01.11.15",
			newDescription: "",
			wantErr:        true,
			errMsg:         "description is required",
		},
		{
			name:           "whitespace description",
			id:             "S01.11.15",
			newDescription: "   ",
			wantErr:        true,
			errMsg:         "description is required",
		},
		{
			name:           "scope cannot be renamed",
			id:             "S01",
			newDescription: "New Scope",
			wantErr:        true,
			errMsg:         "cannot rename scopes",
		},
		{
			name:           "invalid ID",
			id:             "invalid",
			newDescription: "Name",
			wantErr:        true,
			errMsg:         "invalid ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RenameCommand{
				ID:             tt.id,
				NewDescription: tt.newDescription,
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

func TestRenameEligibility(t *testing.T) {
	tests := []struct {
		name      string
		nodeType  domain.IDType
		canRename bool
	}{
		{name: "item can be renamed", nodeType: domain.IDTypeItem, canRename: true},
		{name: "category can be renamed", nodeType: domain.IDTypeCategory, canRename: true},
		{name: "area can be renamed", nodeType: domain.IDTypeArea, canRename: true},
		{name: "scope cannot be renamed", nodeType: domain.IDTypeScope, canRename: false},
		{name: "file cannot be renamed", nodeType: domain.IDTypeFile, canRename: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckRenameEligibility(tt.nodeType)
			if result.CanRename != tt.canRename {
				t.Errorf("expected canRename=%v, got %v (reason: %s)", tt.canRename, result.CanRename, result.Reason)
			}
		})
	}
}
