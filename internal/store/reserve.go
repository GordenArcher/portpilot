package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

var ErrReservationNotFound = errors.New("reservation not found")

const (
	configDirName      = ".portpilot"
	reservationsFile   = "reservations.json"
	reservationsTmpExt = ".tmp"
)

// LoadReservations reads the user's reserved port labels from disk.
//
// The rest of the CLI treats reservations as optional metadata layered on top
// of the live OS scan. That means a new user should be able to run
// `portpilot scan` before any config file exists, so a missing reservations
// file is intentionally returned as an empty map instead of an error.
func LoadReservations() (map[int]string, error) {
	path, err := ReservationsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[int]string{}, nil
		}
		return nil, fmt.Errorf("read reservations file: %w", err)
	}

	if len(data) == 0 {
		return map[int]string{}, nil
	}

	raw := map[string]string{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse reservations file: %w", err)
	}

	reservations := make(map[int]string, len(raw))
	for key, label := range raw {
		port, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("parse reserved port %q: %w", key, err)
		}
		reservations[port] = label
	}

	return reservations, nil
}

// SaveReservation stores or replaces the label for one port.
//
// I load the existing map first because reservations are a user maintained
// collection. Writing only the new port would silently discard every existing
// label, which is the kind of data loss that makes a small CLI feel unreliable.
func SaveReservation(port int, label string) error {
	reservations, err := LoadReservations()
	if err != nil {
		return err
	}

	reservations[port] = label
	return SaveReservations(reservations)
}

// DeleteReservation removes one saved port label without disturbing the rest
// of the reservation file.
//
// I return ErrReservationNotFound instead of treating a missing key as success
// because the CLI should be honest with the user. If they try to unreserve the
// wrong port, silently rewriting the file would hide that mistake and make it
// harder to understand why future scans still show another reserved port.
func DeleteReservation(port int) (string, error) {
	reservations, err := LoadReservations()
	if err != nil {
		return "", err
	}

	label, ok := reservations[port]
	if !ok {
		return "", ErrReservationNotFound
	}

	delete(reservations, port)
	return label, SaveReservations(reservations)
}

// SaveReservations writes the full reservation map to disk as stable JSON.
//
// Ports are represented as object keys because JSON cannot encode integer map
// keys directly in a portable way. Keeping the public API as map[int]string
// still lets command code work with ports as actual numbers instead of leaking
// serialization details into the CLI layer.
func SaveReservations(reservations map[int]string) error {
	path, err := ReservationsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create portpilot config directory: %w", err)
	}

	raw := make(map[string]string, len(reservations))
	for port, label := range reservations {
		raw[strconv.Itoa(port)] = label
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("encode reservations: %w", err)
	}
	data = append(data, '\n')

	tmpPath := path + reservationsTmpExt
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temporary reservations file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace reservations file: %w", err)
	}

	return nil
}

// ReservationsPath centralizes where portpilot keeps reservation data so every
// command uses the same file and future commands such as unreserve/export do
// not accidentally create competing storage locations.
func ReservationsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, configDirName, reservationsFile), nil
}
