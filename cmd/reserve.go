package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GordenArcher/portpilot/internal/store"
	"github.com/spf13/cobra"
)

var reserveCmd = &cobra.Command{
	Use:     "reserve <port> <label>",
	Short:   "Tag a port with a local label",
	Args:    cobra.ExactArgs(2),
	Example: `  portpilot reserve 8080 "axon-core"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %q must be a number", args[0])
		}

		if err := validatePort(port); err != nil {
			return err
		}

		label := strings.TrimSpace(args[1])
		if label == "" {
			return fmt.Errorf("reservation label cannot be empty")
		}

		// A reservation is intentionally just metadata. It does not bind the
		// port or stop another process from using it; it gives future scans
		// enough context to explain why a free or occupied port matters to the
		// user.
		if err := store.SaveReservation(port, label); err != nil {
			return fmt.Errorf("failed to reserve port %d: %w", port, err)
		}

		fmt.Printf("Reserved port %d as %q\n", port, label)
		return nil
	},
}
