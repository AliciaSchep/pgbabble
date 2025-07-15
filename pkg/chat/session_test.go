package chat

import (
	"testing"

	"github.com/AliciaSchep/pgbabble/pkg/db"
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

