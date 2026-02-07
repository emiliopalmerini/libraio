package application

import (
	"errors"
	"testing"

	"libraio/internal/domain"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantErr   bool
	}{
		{
			name:      "valid value",
			fieldName: "description",
			value:     "Test Description",
			wantErr:   false,
		},
		{
			name:      "empty string",
			fieldName: "description",
			value:     "",
			wantErr:   true,
		},
		{
			name:      "whitespace only",
			fieldName: "description",
			value:     "   ",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.fieldName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("expected ValidationError, got %T", err)
				}
				if valErr.Field != tt.fieldName {
					t.Errorf("expected field %s, got %s", tt.fieldName, valErr.Field)
				}
			}
		})
	}
}

func TestValidateIDType(t *testing.T) {
	tests := []struct {
		name         string
		fieldName    string
		id           string
		expectedType domain.IDType
		wantErr      bool
	}{
		{
			name:         "valid scope ID",
			fieldName:    "scopeID",
			id:           "S01",
			expectedType: domain.IDTypeScope,
			wantErr:      false,
		},
		{
			name:         "invalid scope ID - wrong type",
			fieldName:    "scopeID",
			id:           "S01.10-19",
			expectedType: domain.IDTypeScope,
			wantErr:      true,
		},
		{
			name:         "valid area ID",
			fieldName:    "areaID",
			id:           "S01.10-19",
			expectedType: domain.IDTypeArea,
			wantErr:      false,
		},
		{
			name:         "valid category ID",
			fieldName:    "categoryID",
			id:           "S01.11",
			expectedType: domain.IDTypeCategory,
			wantErr:      false,
		},
		{
			name:         "valid item ID",
			fieldName:    "itemID",
			id:           "S01.11.15",
			expectedType: domain.IDTypeItem,
			wantErr:      false,
		},
		{
			name:         "invalid ID format",
			fieldName:    "categoryID",
			id:           "invalid",
			expectedType: domain.IDTypeCategory,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIDType(tt.fieldName, tt.id, tt.expectedType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIDType() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("expected ValidationError, got %T", err)
				}
				if valErr.Field != tt.fieldName {
					t.Errorf("expected field %s, got %s", tt.fieldName, valErr.Field)
				}
			}
		})
	}
}
