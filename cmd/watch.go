package cmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gordenarcher/portpilot/internal/notify"
	"github.com/gordenarcher/portpilot/internal/ports"
	"github.com/spf13/cobra"
)

type portStatusFunc func(int) (bool, error)
type portNotifyFunc func(int, bool) error

var watchCmd = &cobra.Command{
	Use:     "watch <port>",
	Short:   "Watch a port and get notified when its status changes",
	Args:    cobra.ExactArgs(1),
	Example: `  portpilot watch 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %q must be a number", args[0])
		}

		if err := validatePort(port); err != nil {
			return err
		}

		return runWatch(context.Background(), cmd.OutOrStdout(), port, time.Second, ports.IsOccupied, notifyPortChange)
	},
}

func runWatch(
	ctx context.Context,
	output io.Writer,
	port int,
	interval time.Duration,
	isOccupied portStatusFunc,
	notifyChange portNotifyFunc,
) error {
	fmt.Fprintf(output, "Watching port %d. Press Ctrl+C to stop\n\n", port)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastOccupied *bool
	for {
		hadPreviousStatus := lastOccupied != nil
		changed, occupied, err := checkWatchedPort(port, isOccupied, &lastOccupied)
		if err != nil {
			return err
		}

		if changed {
			printWatchStatus(output, port, occupied)
			if hadPreviousStatus {
				if err := notifyChange(port, occupied); err != nil {
					fmt.Fprintf(output, "Notification failed: %v\n", err)
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func checkWatchedPort(port int, isOccupied portStatusFunc, lastOccupied **bool) (bool, bool, error) {
	occupied, err := isOccupied(port)
	if err != nil {
		return false, false, fmt.Errorf("watch error on port %d: %w", port, err)
	}

	if *lastOccupied == nil {
		*lastOccupied = &occupied
		return true, occupied, nil
	}

	if **lastOccupied == occupied {
		return false, occupied, nil
	}

	*lastOccupied = &occupied
	return true, occupied, nil
}

func printWatchStatus(output io.Writer, port int, occupied bool) {
	timestamp := time.Now().Format("15:04:05")
	if occupied {
		fmt.Fprintf(output, "[%s] Port %d is now OCCUPIED\n", timestamp, port)
		return
	}

	fmt.Fprintf(output, "[%s] Port %d is now FREE\n", timestamp, port)
}

func notifyPortChange(port int, occupied bool) error {
	status := "FREE"
	if occupied {
		status = "OCCUPIED"
	}

	return notify.Send("portpilot", fmt.Sprintf("Port %d is now %s", port, status))
}
