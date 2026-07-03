package export

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GordenArcher/portpilot/internal/ports"
)

func TestNewReportSortsRecordsAndSummarizesReservations(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 30, 0, 0, time.FixedZone("test", 2*60*60))
	results := []ports.PortInfo{
		{Port: 9000, PID: 90, Process: "worker", State: "LISTEN"},
		{Port: 3000, PID: 30, Process: "web", State: "LISTEN"},
		{Port: 3000, PID: 20, Process: "web-helper", State: "LISTEN"},
	}
	reservations := map[int]string{
		9000: "jobs",
		7000: "axon-backend",
	}

	report := NewReport(now, "3000-9000", results, reservations)

	if !report.GeneratedAt.Equal(now.UTC()) {
		t.Fatalf("GeneratedAt should be normalized to UTC, got %s", report.GeneratedAt)
	}
	if report.Filter != "3000-9000" {
		t.Fatalf("filter mismatch: %q", report.Filter)
	}

	wantSummary := Summary{
		Listening:           3,
		Reserved:            2,
		FreeReservations:    1,
		MatchedReservations: 1,
	}
	if report.Summary != wantSummary {
		t.Fatalf("summary mismatch\nwant: %#v\n got: %#v", wantSummary, report.Summary)
	}

	wantRecords := []PortRecord{
		{Port: 3000, PID: 20, Process: "web-helper", State: "LISTEN", Status: "occupied"},
		{Port: 3000, PID: 30, Process: "web", State: "LISTEN", Status: "occupied"},
		{Port: 9000, PID: 90, Process: "worker", State: "LISTEN", Status: "reserved", Reservation: "jobs"},
		{Port: 7000, State: "FREE", Status: "free_reserved", Reservation: "axon-backend"},
	}
	if !reflect.DeepEqual(report.Ports, wantRecords) {
		t.Fatalf("records mismatch\nwant: %#v\n got: %#v", wantRecords, report.Ports)
	}
}

func TestNewReportHandlesEmptyScanWithOnlyReservations(t *testing.T) {
	report := NewReport(time.Unix(0, 0), "", nil, map[int]string{
		8080: "api",
		3000: "web",
	})

	wantSummary := Summary{
		Listening:           0,
		Reserved:            2,
		FreeReservations:    2,
		MatchedReservations: 0,
	}
	if report.Summary != wantSummary {
		t.Fatalf("summary mismatch\nwant: %#v\n got: %#v", wantSummary, report.Summary)
	}

	wantPorts := []PortRecord{
		{Port: 3000, State: "FREE", Status: "free_reserved", Reservation: "web"},
		{Port: 8080, State: "FREE", Status: "free_reserved", Reservation: "api"},
	}
	if !reflect.DeepEqual(report.Ports, wantPorts) {
		t.Fatalf("ports mismatch\nwant: %#v\n got: %#v", wantPorts, report.Ports)
	}
}

func TestWriteJSONProducesIndentedValidJSONWithNewline(t *testing.T) {
	report := NewReport(time.Unix(1, 0), "7000-7000", []ports.PortInfo{
		{Port: 7000, PID: 595, Process: "ControlCe", State: "LISTEN"},
	}, map[int]string{7000: "axon-backend"})

	var buf bytes.Buffer
	if err := WriteJSON(&buf, report); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\n  \"summary\": {") {
		t.Fatalf("expected indented JSON, got:\n%s", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("JSON output should end with newline, got %q", output)
	}

	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output should be valid JSON: %v\n%s", err, output)
	}
	if decoded.Summary.MatchedReservations != 1 {
		t.Fatalf("decoded matched reservations mismatch: %#v", decoded.Summary)
	}
}
