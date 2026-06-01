package appshell

import (
	"strings"
	"testing"
)

func TestNewCorrelationID(t *testing.T) {
	id := NewCorrelationID()

	if id == "" {
		t.Fatal("expected non-empty correlation ID")
	}
	// Format: cp-<hex>-<nanos>
	if len(id) < 10 {
		t.Errorf("ID too short: %s", id)
	}
	if id[:3] != "cp-" {
		t.Errorf("expected prefix 'cp-', got: %s", id[:3])
	}
}

func TestNewCorrelationID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewCorrelationID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestNewCorrelationID_Format(t *testing.T) {
	id := NewCorrelationID()

	// Must have three segments: cp / uuid / nano
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts separated by '-', got %d parts: %v", len(parts), parts)
	}
	// UUID part must be 32 hex chars
	if len(parts[1]) != 32 {
		t.Errorf("expected UUID part to be 32 hex chars, got %d: %q", len(parts[1]), parts[1])
	}
	for _, c := range parts[1] {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("UUID part contains non-hex char: %c in %q", c, parts[1])
		}
	}
	// Nano part must be non-empty and numeric
	if parts[2] == "" {
		t.Errorf("nano part should not be empty")
	}
	for _, c := range parts[2] {
		if c < '0' || c > '9' {
			t.Errorf("nano part contains non-digit: %c in %q", c, parts[2])
		}
	}
}