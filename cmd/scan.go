package cmd

import (
	"fmt"
	"io"
	"time"

	reportjson "github.com/GordenArcher/portpilot/internal/export"
	"github.com/GordenArcher/portpilot/internal/ports"
	"github.com/GordenArcher/portpilot/internal/store"
	"github.com/GordenArcher/portpilot/internal/ui"
	"github.com/spf13/cobra"
)

var filterRange string
var scanJSON bool

type scanPortsFunc func(string) ([]ports.PortInfo, error)
type scanReservationsFunc func() (map[int]string, error)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "List all occupied ports with process info",
	Example: `  portpilot scan
  portpilot scan --filter 3000-9000
  portpilot scan --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScan(cmd.OutOrStdout(), filterRange, scanJSON, time.Now, ports.Scan, store.LoadReservations)
	},
}

func init() {
	scanCmd.Flags().StringVarP(&filterRange, "filter", "f", "", "Port range to scan, for example 3000-9000")
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "Print scan results as JSON for scripts")
}

func runScan(
	output io.Writer,
	filter string,
	asJSON bool,
	now func() time.Time,
	scan scanPortsFunc,
	loadReservations scanReservationsFunc,
) error {
	if asJSON {
		return runScanJSON(output, filter, now, scan, loadReservations)
	}

	var results []ports.PortInfo
	var reservations map[int]string

	err := ui.WithLoading("Scanning local ports", func() error {
		var err error
		results, err = scan(filter)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		// Enrich scan results with any reserved port labels before rendering.
		// Reservations are user defined and stored locally. They do not affect
		// the scan itself, just the display.
		reservations, err = loadReservations()
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
}

func runScanJSON(
	output io.Writer,
	filter string,
	now func() time.Time,
	scan scanPortsFunc,
	loadReservations scanReservationsFunc,
) error {
	results, err := scan(filter)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	reservations, err := loadReservations()
	if err != nil {
		return fmt.Errorf("failed to load reservations for JSON scan: %w", err)
	}

	report := reportjson.NewReport(now(), filter, results, reservations)
	return reportjson.WriteJSON(output, report)
}
