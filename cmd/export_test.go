package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reportjson "github.com/GordenArcher/portpilot/internal/export"
	"github.com/GordenArcher/portpilot/internal/ports"
)

func TestRunExportWritesJSONToStdoutWithoutStatusText(t *testing.T) {
	var out bytes.Buffer
	fixedTime := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)

	err := runExport(
		&out,
		"7000-7000",
		"",
		func() time.Time { return fixedTime },
		func(filter string) ([]ports.PortInfo, error) {
			if filter != "7000-7000" {
				t.Fatalf("filter passed to scan mismatch: %q", filter)
			}
			return []ports.PortInfo{{Port: 7000, PID: 595, Process: "ControlCe", State: "LISTEN"}}, nil
		},
		func() (map[int]string, error) {
			return map[int]string{7000: "axon-backend"}, nil
		},
	)
	if err != nil {
		t.Fatalf("runExport: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "Exported scan results") {
		t.Fatalf("stdout JSON mode should not include status text:\n%s", output)
	}

	var report reportjson.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\n%s", err, output)
	}
	if report.GeneratedAt != fixedTime {
		t.Fatalf("generated time mismatch: %s", report.GeneratedAt)
	}
	if report.Summary.MatchedReservations != 1 {
		t.Fatalf("expected one matched reservation, got %#v", report.Summary)
	}
}

func TestRunExportWritesJSONFileAndCreatesParentDirectory(t *testing.T) {
	var out bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "nested", "ports.json")

	err := runExport(
		&out,
		"",
		outputPath,
		func() time.Time { return time.Unix(1, 0) },
		func(string) ([]ports.PortInfo, error) {
			return nil, nil
		},
		func() (map[int]string, error) {
			return map[int]string{8080: "api"}, nil
		},
	)
	if err != nil {
		t.Fatalf("runExport file mode: %v", err)
	}

	if !strings.Contains(out.String(), "Exported scan results to "+outputPath) {
		t.Fatalf("expected file mode status output, got %q", out.String())
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}

	var report reportjson.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("export file should contain valid JSON: %v\n%s", err, string(data))
	}
	if report.Summary.FreeReservations != 1 {
		t.Fatalf("expected free reservation in export file, got %#v", report.Summary)
	}
}

func TestRunExportWrapsScanAndReservationErrors(t *testing.T) {
	t.Run("scan error", func(t *testing.T) {
		err := runExport(
			&bytes.Buffer{},
			"",
			"",
			time.Now,
			func(string) ([]ports.PortInfo, error) {
				return nil, errors.New("scan broke")
			},
			func() (map[int]string, error) {
				t.Fatal("reservations should not load after scan failure")
				return nil, nil
			},
		)
		if err == nil || !strings.Contains(err.Error(), "export scan failed") {
			t.Fatalf("expected wrapped scan error, got %v", err)
		}
	})

	t.Run("reservation error", func(t *testing.T) {
		err := runExport(
			&bytes.Buffer{},
			"",
			"",
			time.Now,
			func(string) ([]ports.PortInfo, error) {
				return nil, nil
			},
			func() (map[int]string, error) {
				return nil, errors.New("bad reservations")
			},
		)
		if err == nil || !strings.Contains(err.Error(), "failed to load reservations") {
			t.Fatalf("expected wrapped reservation error, got %v", err)
		}
	})
}
