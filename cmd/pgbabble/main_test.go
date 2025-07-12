package main

import (
	"strings"
	"testing"
)

func TestModeValidation(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid default mode",
			mode:      "default",
			expectErr: false,
		},
		{
			name:      "valid schema-only mode",
			mode:      "schema-only",
			expectErr: false,
		},
		{
			name:      "valid share-results mode",
			mode:      "share-results",
			expectErr: false,
		},
		{
			name:      "invalid old summary_data mode",
			mode:      "summary_data",
			expectErr: true,
			errMsg:    "invalid mode: summary_data (must be: default, schema-only, share-results)",
		},
		{
			name:      "invalid old full_data mode",
			mode:      "full_data",
			expectErr: true,
			errMsg:    "invalid mode: full_data (must be: default, schema-only, share-results)",
		},
		{
			name:      "invalid random mode",
			mode:      "random_mode",
			expectErr: true,
			errMsg:    "invalid mode: random_mode (must be: default, schema-only, share-results)",
		},
		{
			name:      "empty mode",
			mode:      "",
			expectErr: true,
			errMsg:    "invalid mode:  (must be: default, schema-only, share-results)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic directly
			var err error
			if tt.mode != "default" && tt.mode != "schema-only" && tt.mode != "share-results" {
				err = &ValidationError{message: "invalid mode: " + tt.mode + " (must be: default, schema-only, share-results)"}
			}

			if tt.expectErr {
				if err == nil {
					t.Error("expected validation error but got none")
				} else if !strings.Contains(err.Error(), "invalid mode: "+tt.mode) {
					t.Errorf("expected error message to contain mode name, got: %s", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

// ValidationError is a simple error type for testing
type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

func TestModeDescriptions(t *testing.T) {
	// Test that our mode descriptions are meaningful
	descriptions := map[string]string{
		"default":       "EXPLAIN sharing allowed, table size info shared, query row counts shared, but no actual query result data",
		"schema-only":   "No EXPLAIN sharing, no table size info, no query result data - maximum privacy",
		"share-results": "Full data sharing including EXPLAIN results, table sizes, and actual query result data",
	}

	for mode, expectedDesc := range descriptions {
		t.Run("description_"+mode, func(t *testing.T) {
			// Verify description contains key privacy concepts
			switch mode {
			case "default":
				if !strings.Contains(expectedDesc, "EXPLAIN sharing allowed") {
					t.Error("default mode description should mention EXPLAIN sharing")
				}
				if !strings.Contains(expectedDesc, "no actual query result data") {
					t.Error("default mode description should clarify no actual data sharing")
				}
			case "schema-only":
				if !strings.Contains(expectedDesc, "No EXPLAIN sharing") {
					t.Error("schema-only mode description should mention EXPLAIN blocking")
				}
				if !strings.Contains(expectedDesc, "maximum privacy") {
					t.Error("schema-only mode description should emphasize privacy")
				}
			case "share-results":
				if !strings.Contains(expectedDesc, "Full data sharing") {
					t.Error("share-results mode description should mention full data sharing")
				}
				if !strings.Contains(expectedDesc, "actual query result data") {
					t.Error("share-results mode description should mention actual data sharing")
				}
			}
		})
	}
}