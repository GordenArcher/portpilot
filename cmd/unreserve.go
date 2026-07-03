package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/GordenArcher/portpilot/internal/store"
	"github.com/spf13/cobra"
)

var unreserveCmd = &cobra.Command{
	Use:     "unreserve <port>",
	Short:   "Remove a saved port reservation",
	Args:    cobra.ExactArgs(1),
	Example: `  portpilot unreserve 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %q must be a number", args[0])
		}

		if err := validatePort(port); err != nil {
			return err
		}

		label, err := store.DeleteReservation(port)
		if err != nil {
			if errors.Is(err, store.ErrReservationNotFound) {
				return fmt.Errorf("port %d is not reserved", port)
			}
			return fmt.Errorf("failed to unreserve port %d: %w", port, err)
		}

		fmt.Printf("Removed reservation for port %d (%q)\n", port, label)
		return nil
	},
}
