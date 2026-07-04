package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	reportjson "github.com/GordenArcher/portpilot/internal/export"
	"github.com/GordenArcher/portpilot/internal/ports"
	"github.com/GordenArcher/portpilot/internal/store"
	"github.com/GordenArcher/portpilot/internal/ui"
	"github.com/spf13/cobra"
)

var filterRange string
var scanJSON bool
var scanWatch bool
var scanWatchInterval time.Duration

type scanPortsFunc func(string) ([]ports.PortInfo, error)
type scanReservationsFunc func() (map[int]string, error)
type scanRenderFunc func([]ports.PortInfo, map[int]string)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "List all occupied ports with process info",
	Example: `  portpilot scan
  portpilot scan --filter 3000-9000
  portpilot scan --json
  portpilot scan --watch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if scanWatch && scanJSON {
			return fmt.Errorf("scan --watch and --json cannot be used together")
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		return runScan(cmd.OutOrStdout(), filterRange, scanJSON, scanWatch, scanWatchInterval, time.Now, ports.Scan, store.LoadReservations, ui.RenderTable, ctx)
	},
}

func init() {
	scanCmd.Flags().StringVarP(&filterRange, "filter", "f", "", "Port range to scan, for example 3000-9000")
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "Print scan results as JSON for scripts")
	scanCmd.Flags().BoolVar(&scanWatch, "watch", false, "Continuously refresh scan results")
	scanCmd.Flags().DurationVar(&scanWatchInterval, "interval", 2*time.Second, "Refresh interval for scan --watch")
}

func runScan(
	output io.Writer,
	filter string,
	asJSON bool,
	watch bool,
	interval time.Duration,
	now func() time.Time,
	scan scanPortsFunc,
	loadReservations scanReservationsFunc,
	render scanRenderFunc,
	ctx context.Context,
) error {
	if asJSON {
		return runScanJSON(output, filter, now, scan, loadReservations)
	}
	if watch {
		return runScanWatch(ctx, output, filter, interval, scan, loadReservations, render)
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

	render(results, reservations)
	return nil
}

// runScanJSON is intentionally separate from the styled scan path because JSON
// output is usually consumed by scripts, jq, or CI. Any spinner, dashboard
// border, or warning text on stdout would corrupt the stream and make the flag
// unreliable for automation.
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

// runScanWatch keeps re-rendering the normal scan dashboard until the context
// is canceled. The command passes a signal-aware context so Ctrl+C exits the
// loop cleanly, while tests can pass a manually canceled context and verify the
// refresh behavior without sleeping forever.
func runScanWatch(
	ctx context.Context,
	output io.Writer,
	filter string,
	interval time.Duration,
	scan scanPortsFunc,
	loadReservations scanReservationsFunc,
	render scanRenderFunc,
) error {
	if interval <= 0 {
		return fmt.Errorf("scan watch interval must be greater than zero")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := renderScanWatchFrame(output, filter, scan, loadReservations, render); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// renderScanWatchFrame performs one complete refresh cycle.
//
// Reservation load failures stay non-fatal here for the same reason they are
// non-fatal in the regular table scan: live port visibility is still useful
// even when the user's metadata file is temporarily missing or corrupt.
func renderScanWatchFrame(
	output io.Writer,
	filter string,
	scan scanPortsFunc,
	loadReservations scanReservationsFunc,
	render scanRenderFunc,
) error {
	results, err := scan(filter)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	reservations, err := loadReservations()
	if err != nil {
		reservations = map[int]string{}
	}

	fmt.Fprint(output, "\033[H\033[2J")
	fmt.Fprintf(output, "Live scan refresh. Press Ctrl+C to stop. Updated %s\n", time.Now().Format("15:04:05"))
	render(results, reservations)
	return nil
}
