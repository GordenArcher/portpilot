package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadReservationsMissingFileReturnsEmptyMap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	reservations, err := LoadReservations()
	if err != nil {
		t.Fatalf("LoadReservations returned error for a first run user: %v", err)
	}

	if len(reservations) != 0 {
		t.Fatalf("expected no reservations, got %#v", reservations)
	}
}

func TestSaveReservationCreatesStableJSONAndPreservesExistingPorts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := SaveReservation(7000, "axon-backend"); err != nil {
		t.Fatalf("SaveReservation first port: %v", err)
	}
	if err := SaveReservation(3000, "web-ui"); err != nil {
		t.Fatalf("SaveReservation second port: %v", err)
	}

	got, err := LoadReservations()
	if err != nil {
		t.Fatalf("LoadReservations after save: %v", err)
	}

	want := map[int]string{
		7000: "axon-backend",
		3000: "web-ui",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reservations mismatch\nwant: %#v\n got: %#v", want, got)
	}

	path := filepath.Join(home, configDirName, reservationsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reservations file: %v", err)
	}

	content := string(data)
	for _, expected := range []string{`"3000": "web-ui"`, `"7000": "axon-backend"`} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in reservation file, got:\n%s", expected, content)
		}
	}
	if !strings.HasSuffix(content, "\n") {
		t.Fatalf("reservation file should end with newline for clean diffs, got %q", content)
	}
}

func TestDeleteReservationRemovesOnlyRequestedPort(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := SaveReservations(map[int]string{
		7000: "axon-backend",
		8080: "api",
	}); err != nil {
		t.Fatalf("SaveReservations: %v", err)
	}

	label, err := DeleteReservation(7000)
	if err != nil {
		t.Fatalf("DeleteReservation: %v", err)
	}
	if label != "axon-backend" {
		t.Fatalf("deleted label mismatch: %q", label)
	}

	got, err := LoadReservations()
	if err != nil {
		t.Fatalf("LoadReservations after delete: %v", err)
	}

	want := map[int]string{8080: "api"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reservations mismatch after delete\nwant: %#v\n got: %#v", want, got)
	}
}

func TestDeleteReservationMissingPortReturnsTypedErrorAndKeepsFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	original := map[int]string{7000: "axon-backend"}
	if err := SaveReservations(original); err != nil {
		t.Fatalf("SaveReservations: %v", err)
	}

	_, err := DeleteReservation(3000)
	if !errors.Is(err, ErrReservationNotFound) {
		t.Fatalf("expected ErrReservationNotFound, got %v", err)
	}

	got, err := LoadReservations()
	if err != nil {
		t.Fatalf("LoadReservations after failed delete: %v", err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("failed delete should not mutate reservations\nwant: %#v\n got: %#v", original, got)
	}
}

func TestLoadReservationsRejectsCorruptJSONAndInvalidPortKeys(t *testing.T) {
	t.Run("corrupt json", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		path := filepath.Join(home, configDirName, reservationsFile)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(`{"7000":`), 0o644); err != nil {
			t.Fatalf("write corrupt json: %v", err)
		}

		if _, err := LoadReservations(); err == nil || !strings.Contains(err.Error(), "parse reservations file") {
			t.Fatalf("expected parse error for corrupt json, got %v", err)
		}
	})

	t.Run("invalid port key", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		path := filepath.Join(home, configDirName, reservationsFile)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(`{"not-a-port":"bad"}`), 0o644); err != nil {
			t.Fatalf("write invalid key json: %v", err)
		}

		if _, err := LoadReservations(); err == nil || !strings.Contains(err.Error(), "parse reserved port") {
			t.Fatalf("expected invalid port key error, got %v", err)
		}
	})
}
