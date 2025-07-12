package chat

import (
	"testing"

	"pgbabble/pkg/db"
)

func TestNewSession(t *testing.T) {
	// Mock connection (nil for this test)
	var conn *db.Connection
	mode := "default"

	session := NewSession(conn, mode)

	if session.conn != conn {
		t.Error("expected connection to be set correctly")
	}

	if session.mode != mode {
		t.Errorf("expected mode %s, got %s", mode, session.mode)
	}
}

func TestSession_setMode(t *testing.T) {
	session := &Session{mode: "default"}

	tests := []struct {
		name        string
		newMode     string
		expectError bool
	}{
		{
			name:        "valid mode - schema-only",
			newMode:     "schema-only",
			expectError: false,
		},
		{
			name:        "valid mode - share-results",
			newMode:     "share-results",
			expectError: false,
		},
		{
			name:        "valid mode - default",
			newMode:     "default",
			expectError: false,
		},
		{
			name:        "invalid mode",
			newMode:     "invalid_mode",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMode := session.mode
			err := session.setMode(tt.newMode)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				// Mode should not change on error
				if session.mode != originalMode {
					t.Errorf("mode should not change on error, expected %s, got %s", originalMode, session.mode)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if session.mode != tt.newMode {
					t.Errorf("expected mode %s, got %s", tt.newMode, session.mode)
				}
			}
		})
	}
}