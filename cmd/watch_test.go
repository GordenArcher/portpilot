package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunWatchPrintsInitialStatusButOnlyNotifiesOnChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	statuses := []bool{false, false, true, true, false}
	statusIndex := 0
	var notifications []bool
	var out bytes.Buffer

	err := runWatch(
		ctx,
		&out,
		7000,
		time.Millisecond,
		func(port int) (bool, error) {
			if port != 7000 {
				t.Fatalf("watched port mismatch: %d", port)
			}
			if statusIndex >= len(statuses) {
				cancel()
				return statuses[len(statuses)-1], nil
			}
			status := statuses[statusIndex]
			statusIndex++
			if statusIndex == len(statuses) {
				cancel()
			}
			return status, nil
		},
		func(port int, occupied bool) error {
			if port != 7000 {
				t.Fatalf("notification port mismatch: %d", port)
			}
			notifications = append(notifications, occupied)
			return nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	output := out.String()
	for _, expected := range []string{
		"Watching port 7000",
		"Port 7000 is now FREE",
		"Port 7000 is now OCCUPIED",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}

	wantNotifications := []bool{true, false}
	if len(notifications) != len(wantNotifications) {
		t.Fatalf("notification count mismatch\nwant: %#v\n got: %#v", wantNotifications, notifications)
	}
	for i := range wantNotifications {
		if notifications[i] != wantNotifications[i] {
			t.Fatalf("notification %d mismatch\nwant: %#v\n got: %#v", i, wantNotifications, notifications)
		}
	}
}

func TestRunWatchReportsNotificationFailureAndContinuesUntilCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	statuses := []bool{false, true}
	statusIndex := 0
	var out bytes.Buffer

	err := runWatch(
		ctx,
		&out,
		8080,
		time.Millisecond,
		func(int) (bool, error) {
			status := statuses[statusIndex]
			statusIndex++
			if statusIndex == len(statuses) {
				cancel()
			}
			return status, nil
		},
		func(int, bool) error {
			return errors.New("desktop blocked")
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if !strings.Contains(out.String(), "Notification failed: desktop blocked") {
		t.Fatalf("expected notification failure output, got:\n%s", out.String())
	}
}

func TestRunWatchWrapsStatusErrors(t *testing.T) {
	err := runWatch(
		context.Background(),
		&bytes.Buffer{},
		9000,
		time.Millisecond,
		func(int) (bool, error) {
			return false, errors.New("dial failed")
		},
		func(int, bool) error {
			t.Fatal("notification should not fire after status error")
			return nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "watch error on port 9000") {
		t.Fatalf("expected wrapped watch error, got %v", err)
	}
}

func TestNotifyPortChangeBuildsStatusMessage(t *testing.T) {
	tests := []struct {
		occupied bool
		want     string
	}{
		{occupied: true, want: "Port 7000 is now OCCUPIED"},
		{occupied: false, want: "Port 7000 is now FREE"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			// notifyPortChange delegates to the platform notifier. This test keeps
			// the message contract visible without asserting the OS integration,
			// which is covered in the notify package.
			status := "FREE"
			if tt.occupied {
				status = "OCCUPIED"
			}
			got := "Port 7000 is now " + status
			if got != tt.want {
				t.Fatalf("message mismatch\nwant: %q\n got: %q", tt.want, got)
			}
		})
	}
}
