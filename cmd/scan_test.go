package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	reportjson "github.com/GordenArcher/portpilot/internal/export"
	"github.com/GordenArcher/portpilot/internal/ports"
)

func TestRunScanJSONWritesMachineReadableReport(t *testing.T) {
	var out bytes.Buffer
	fixedTime := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)

	err := runScan(
		&out,
		"7000-7000",
		true,
		false,
		time.Second,
		func() time.Time { return fixedTime },
		func(filter string) ([]ports.PortInfo, error) {
			if filter != "7000-7000" {
				t.Fatalf("filter mismatch: %q", filter)
			}
			return []ports.PortInfo{{Port: 7000, PID: 595, Process: "ControlCe", State: "LISTEN"}}, nil
		},
		func() (map[int]string, error) {
			return map[int]string{7000: "axon-backend"}, nil
		},
		noopScanRender,
		context.Background(),
	)
	if err != nil {
		t.Fatalf("runScan JSON: %v", err)
	}

	output := out.String()
	if strings.Contains(output, "Scanning local ports") {
		t.Fatalf("JSON scan must not include loading text:\n%s", output)
	}

	var report reportjson.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("scan JSON should be valid JSON: %v\n%s", err, output)
	}
	if report.GeneratedAt != fixedTime {
		t.Fatalf("generated time mismatch: %s", report.GeneratedAt)
	}
	if report.Summary.MatchedReservations != 1 {
		t.Fatalf("expected matched reservation, got %#v", report.Summary)
	}
	if len(report.Ports) != 1 || report.Ports[0].Reservation != "axon-backend" {
		t.Fatalf("unexpected ports in JSON report: %#v", report.Ports)
	}
}

func TestRunScanJSONReturnsErrorsInsteadOfRenderingFallbacks(t *testing.T) {
	t.Run("scan error", func(t *testing.T) {
		err := runScan(
			&bytes.Buffer{},
			"",
			true,
			false,
			time.Second,
			time.Now,
			func(string) ([]ports.PortInfo, error) {
				return nil, errors.New("scanner failed")
			},
			func() (map[int]string, error) {
				t.Fatal("reservations should not load after scan failure")
				return nil, nil
			},
			noopScanRender,
			context.Background(),
		)
		if err == nil || !strings.Contains(err.Error(), "scan failed") {
			t.Fatalf("expected wrapped scan error, got %v", err)
		}
	})

	t.Run("reservation error", func(t *testing.T) {
		err := runScan(
			&bytes.Buffer{},
			"",
			true,
			false,
			time.Second,
			time.Now,
			func(string) ([]ports.PortInfo, error) {
				return nil, nil
			},
			func() (map[int]string, error) {
				return nil, errors.New("bad reservation file")
			},
			noopScanRender,
			context.Background(),
		)
		if err == nil || !strings.Contains(err.Error(), "failed to load reservations for JSON scan") {
			t.Fatalf("expected wrapped reservation error, got %v", err)
		}
	})
}

func TestRunScanTableIgnoresReservationLoadErrors(t *testing.T) {
	var out bytes.Buffer

	err := runScan(
		&out,
		"3000-3000",
		false,
		false,
		time.Second,
		time.Now,
		func(string) ([]ports.PortInfo, error) {
			return []ports.PortInfo{{Port: 3000, PID: 12, Process: "node", State: "LISTEN"}}, nil
		},
		func() (map[int]string, error) {
			return nil, errors.New("bad reservation file")
		},
		noopScanRender,
		context.Background(),
	)
	if err != nil {
		t.Fatalf("table scan should ignore reservation load errors: %v", err)
	}
}

func TestRunScanWatchRendersFramesUntilContextStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var out bytes.Buffer
	var renderedResults []ports.PortInfo
	var renderedReservations map[int]string
	scanCalls := 0

	err := runScanWatch(
		ctx,
		&out,
		"7000-7000",
		time.Millisecond,
		func(filter string) ([]ports.PortInfo, error) {
			if filter != "7000-7000" {
				t.Fatalf("filter mismatch: %q", filter)
			}
			scanCalls++
			return []ports.PortInfo{{Port: 7000, PID: 595, Process: "ControlCe", State: "LISTEN"}}, nil
		},
		func() (map[int]string, error) {
			return map[int]string{7000: "axon-backend"}, nil
		},
		func(results []ports.PortInfo, reservations map[int]string) {
			renderedResults = append([]ports.PortInfo(nil), results...)
			renderedReservations = reservations
			cancel()
		},
	)
	if err != nil {
		t.Fatalf("runScanWatch: %v", err)
	}

	if scanCalls != 1 {
		t.Fatalf("expected one scan before cancellation, got %d", scanCalls)
	}
	if !strings.Contains(out.String(), "Live scan refresh") {
		t.Fatalf("expected live scan header, got:\n%s", out.String())
	}
	if len(renderedResults) != 1 || renderedResults[0].Port != 7000 {
		t.Fatalf("rendered results mismatch: %#v", renderedResults)
	}
	if renderedReservations[7000] != "axon-backend" {
		t.Fatalf("rendered reservations mismatch: %#v", renderedReservations)
	}
}

func TestRunScanWatchRejectsInvalidInterval(t *testing.T) {
	err := runScanWatch(
		context.Background(),
		&bytes.Buffer{},
		"",
		0,
		func(string) ([]ports.PortInfo, error) { return nil, nil },
		func() (map[int]string, error) { return nil, nil },
		noopScanRender,
	)
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected interval error, got %v", err)
	}
}

func TestRunScanWatchWrapsScanErrors(t *testing.T) {
	err := runScanWatch(
		context.Background(),
		&bytes.Buffer{},
		"9000-9000",
		time.Millisecond,
		func(string) ([]ports.PortInfo, error) {
			return nil, errors.New("lsof broke")
		},
		func() (map[int]string, error) {
			t.Fatal("reservations should not load after scan failure")
			return nil, nil
		},
		noopScanRender,
	)
	if err == nil || !strings.Contains(err.Error(), "scan failed") {
		t.Fatalf("expected scan failure, got %v", err)
	}
}

func noopScanRender([]ports.PortInfo, map[int]string) {}
