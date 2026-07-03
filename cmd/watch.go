package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gordenarcher/portpilot/internal/ports"
	"github.com/spf13/cobra"
)

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

		fmt.Printf("Watching port %d. Press Ctrl+C to stop\n\n", port)

		// Poll every second and print a message whenever status flips.
		// Simple polling is enough here because OS level socket events would add
		// complexity without making a single port watch command meaningfully better.
		var lastOccupied *bool

		for {
			occupied, err := ports.IsOccupied(port)
			if err != nil {
				return fmt.Errorf("watch error on port %d: %w", port, err)
			}

			if lastOccupied == nil || *lastOccupied != occupied {
				lastOccupied = &occupied
				timestamp := time.Now().Format("15:04:05")

				if occupied {
					fmt.Printf("[%s] Port %d is now OCCUPIED\n", timestamp, port)
				} else {
					fmt.Printf("[%s] Port %d is now FREE\n", timestamp, port)
				}
			}

			time.Sleep(1 * time.Second)
		}
	},
}
