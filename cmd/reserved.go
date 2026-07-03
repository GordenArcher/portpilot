package cmd

import (
	"fmt"

	"github.com/gordenarcher/portpilot/internal/store"
	"github.com/gordenarcher/portpilot/internal/ui"
	"github.com/spf13/cobra"
)

var reservedCmd = &cobra.Command{
	Use:     "reserved",
	Short:   "List all reserved ports",
	Example: `  portpilot reserved`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var reservations map[int]string
		err := ui.WithLoading("Loading reserved ports", func() error {
			var err error
			reservations, err = store.LoadReservations()
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to load reservations: %w", err)
		}

		if len(reservations) == 0 {
			ui.RenderEmpty("Reserved Ports", "No reserved ports. Use `portpilot reserve <port> <label>` to add one.")
			return nil
		}

		ui.RenderReservations(reservations)
		return nil
	},
}
