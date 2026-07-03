package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	reportjson "github.com/gordenarcher/portpilot/internal/export"
	"github.com/gordenarcher/portpilot/internal/ports"
	"github.com/gordenarcher/portpilot/internal/store"
	"github.com/spf13/cobra"
)

type exportScanFunc func(string) ([]ports.PortInfo, error)
type exportReservationsFunc func() (map[int]string, error)

var (
	exportFilter string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export scan results as JSON",
	Example: `  portpilot export
  portpilot export --filter 3000-9000
  portpilot export --output ports.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExport(cmd.OutOrStdout(), exportFilter, exportOutput, time.Now, ports.Scan, store.LoadReservations)
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportFilter, "filter", "f", "", "Port range to export, for example 3000-9000")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Write JSON export to a file instead of stdout")
}

func runExport(
	stdout io.Writer,
	filter string,
	outputPath string,
	now func() time.Time,
	scan exportScanFunc,
	loadReservations exportReservationsFunc,
) error {
	results, err := scan(filter)
	if err != nil {
		return fmt.Errorf("export scan failed: %w", err)
	}

	reservations, err := loadReservations()
	if err != nil {
		return fmt.Errorf("failed to load reservations for export: %w", err)
	}

	report := reportjson.NewReport(now(), filter, results, reservations)
	if outputPath == "" {
		return reportjson.WriteJSON(stdout, report)
	}

	dir := filepath.Dir(outputPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create export directory: %w", err)
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	defer file.Close()

	if err := reportjson.WriteJSON(file, report); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}

	fmt.Fprintf(stdout, "Exported scan results to %s\n", outputPath)
	return nil
}
