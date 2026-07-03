package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gordenarcher/portpilot/internal/ports"
	"github.com/gordenarcher/portpilot/internal/ui"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:     "info <port>",
	Short:   "Show detailed info about what is using a port",
	Args:    cobra.ExactArgs(1),
	Example: `  portpilot info 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %q must be a number", args[0])
		}

		if err := validatePort(port); err != nil {
			return err
		}

		var detail *ports.PortDetail
		portIsFree := false
		err = ui.WithLoading(fmt.Sprintf("Inspecting port %d", port), func() error {
			var err error
			detail, err = ports.Info(port)
			if err != nil {
				if errors.Is(err, ports.ErrPortNotFound) {
					portIsFree = true
					return nil
				}
				return fmt.Errorf("failed to get info for port %d: %w", port, err)
			}

			return nil
		})
		if err != nil {
			return err
		}
		if portIsFree {
			ui.RenderFreePort(port)
			return nil
		}

		ui.RenderDetail(detail)
		return nil
	},
}
