package cmd

import (
	"fmt"

	"github.com/gordenarcher/portpilot/internal/ports"
	"github.com/gordenarcher/portpilot/internal/store"
	"github.com/gordenarcher/portpilot/internal/ui"
	"github.com/spf13/cobra"
)

var filterRange string

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "List all occupied ports with process info",
	Example: `  portpilot scan
  portpilot scan --filter 3000-9000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var results []ports.PortInfo
		var reservations map[int]string

		err := ui.WithLoading("Scanning local ports", func() error {
			var err error
			results, err = ports.Scan(filterRange)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Enrich scan results with any reserved port labels before rendering.
			// Reservations are user defined and stored locally. They do not affect
			// the scan itself, just the display.
			reservations, err = store.LoadReservations()
			if err != nil {
				// Missing or corrupt reservations should not block a scan.
				reservations = map[int]string{}
			}

			return nil
		})
		if err != nil {
			return err
		}

		ui.RenderTable(results, reservations)
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&filterRange, "filter", "f", "", "Port range to scan, for example 3000-9000")
}
