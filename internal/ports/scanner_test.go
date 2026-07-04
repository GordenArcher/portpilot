package ports

import (
	"strings"
	"testing"
)

func TestKillPIDRejectsInvalidPID(t *testing.T) {
	err := KillPID(0)
	if err == nil || !strings.Contains(err.Error(), "invalid PID") {
		t.Fatalf("expected invalid PID error, got %v", err)
	}
}
