package cmd

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/GordenArcher/portpilot/internal/ports"
)

func TestUniqueKillTargetsDeduplicatesByPIDAndSortsByPort(t *testing.T) {
	results := []ports.PortInfo{
		{Port: 9000, PID: 90, Process: "worker"},
		{Port: 3000, PID: 30, Process: "web"},
		{Port: 3001, PID: 30, Process: "web"},
		{Port: 7000, PID: 0, Process: "unknown"},
		{Port: 4000, PID: 40, Process: "api"},
	}

	got := uniqueKillTargets(results)
	want := []killTarget{
		{PID: 30, Port: 3000, Process: "web"},
		{PID: 40, Port: 4000, Process: "api"},
		{PID: 90, Port: 9000, Process: "worker"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targets mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestRunKillAllKillsEachUniquePIDWhenForced(t *testing.T) {
	var out bytes.Buffer
	var killed []int

	err := runKillAll(
		&out,
		strings.NewReader(""),
		"3000-9000",
		true,
		func(filter string) ([]ports.PortInfo, error) {
			if filter != "3000-9000" {
				t.Fatalf("filter mismatch: %q", filter)
			}
			return []ports.PortInfo{
				{Port: 3000, PID: 30, Process: "web"},
				{Port: 3001, PID: 30, Process: "web"},
				{Port: 7000, PID: 70, Process: "api"},
			}, nil
		},
		func(pid int) error {
			killed = append(killed, pid)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runKillAll: %v", err)
	}

	wantKilled := []int{30, 70}
	if !reflect.DeepEqual(killed, wantKilled) {
		t.Fatalf("killed PIDs mismatch\nwant: %#v\n got: %#v", wantKilled, killed)
	}

	output := out.String()
	for _, expected := range []string{"Killed PID 30", "Killed PID 70"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestRunKillAllPromptsAndAbortsWhenNotConfirmed(t *testing.T) {
	var out bytes.Buffer
	var killed bool

	err := runKillAll(
		&out,
		strings.NewReader("n\n"),
		"3000-9000",
		false,
		func(string) ([]ports.PortInfo, error) {
			return []ports.PortInfo{{Port: 3000, PID: 30, Process: "web"}}, nil
		},
		func(int) error {
			killed = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runKillAll: %v", err)
	}
	if killed {
		t.Fatal("killPID should not be called when user aborts")
	}
	if !strings.Contains(out.String(), "Aborted.") {
		t.Fatalf("expected abort output, got:\n%s", out.String())
	}
}

func TestRunKillAllHandlesNoTargetsAndErrors(t *testing.T) {
	t.Run("no killable processes", func(t *testing.T) {
		var out bytes.Buffer
		err := runKillAll(
			&out,
			strings.NewReader(""),
			"3000-9000",
			true,
			func(string) ([]ports.PortInfo, error) {
				return []ports.PortInfo{{Port: 3000, PID: 0, Process: "unknown"}}, nil
			},
			func(int) error {
				t.Fatal("killPID should not run without targets")
				return nil
			},
		)
		if err != nil {
			t.Fatalf("runKillAll: %v", err)
		}
		if !strings.Contains(out.String(), "No killable processes") {
			t.Fatalf("expected no target output, got:\n%s", out.String())
		}
	})

	t.Run("scan error", func(t *testing.T) {
		err := runKillAll(
			&bytes.Buffer{},
			strings.NewReader(""),
			"3000-9000",
			true,
			func(string) ([]ports.PortInfo, error) {
				return nil, errors.New("scan broke")
			},
			func(int) error {
				t.Fatal("killPID should not run after scan error")
				return nil
			},
		)
		if err == nil || !strings.Contains(err.Error(), "scan failed before bulk kill") {
			t.Fatalf("expected scan error, got %v", err)
		}
	})

	t.Run("kill error", func(t *testing.T) {
		err := runKillAll(
			&bytes.Buffer{},
			strings.NewReader(""),
			"3000-9000",
			true,
			func(string) ([]ports.PortInfo, error) {
				return []ports.PortInfo{{Port: 3000, PID: 30, Process: "web"}}, nil
			},
			func(int) error {
				return errors.New("permission denied")
			},
		)
		if err == nil || !strings.Contains(err.Error(), "failed to kill PID 30") {
			t.Fatalf("expected kill error, got %v", err)
		}
	})
}
