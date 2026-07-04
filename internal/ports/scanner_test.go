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

func TestParsePIDLinesParsesOnePIDPerLine(t *testing.T) {
	got, err := parsePIDLines("1341\n55887\n")
	if err != nil {
		t.Fatalf("parsePIDLines: %v", err)
	}

	want := []int{1341, 55887}
	if len(got) != len(want) {
		t.Fatalf("PID count mismatch: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("PID at index %d mismatch: want %d, got %d", i, want[i], got[i])
		}
	}
}

func TestParsePIDLinesIgnoresBlankLines(t *testing.T) {
	got, err := parsePIDLines("\n  55887  \n\n")
	if err != nil {
		t.Fatalf("parsePIDLines: %v", err)
	}

	if len(got) != 1 || got[0] != 55887 {
		t.Fatalf("expected one PID 55887, got %#v", got)
	}
}

func TestParsePIDLinesRejectsBadPIDData(t *testing.T) {
	_, err := parsePIDLines("55887\nnot-a-pid\n")
	if err == nil || !strings.Contains(err.Error(), `parse PID "not-a-pid"`) {
		t.Fatalf("expected parse PID error, got %v", err)
	}
}
