package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GordenArcher/portpilot/internal/store"
)

func TestUnreserveCommandRemovesReservation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := store.SaveReservation(7000, "axon-backend"); err != nil {
		t.Fatalf("SaveReservation: %v", err)
	}

	var out bytes.Buffer
	unreserveCmd.SetOut(&out)
	unreserveCmd.SetErr(&out)
	err := unreserveCmd.RunE(unreserveCmd, []string{"7000"})
	if err != nil {
		t.Fatalf("unreserve command returned error: %v", err)
	}

	reservations, err := store.LoadReservations()
	if err != nil {
		t.Fatalf("LoadReservations: %v", err)
	}
	if _, ok := reservations[7000]; ok {
		t.Fatalf("port 7000 should have been removed, got %#v", reservations)
	}
}

func TestUnreserveCommandRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "non numeric", args: []string{"nope"}, want: "must be a number"},
		{name: "out of range", args: []string{"70000"}, want: "must be between"},
		{name: "missing reservation", args: []string{"7000"}, want: "is not reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			err := unreserveCmd.RunE(unreserveCmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}
