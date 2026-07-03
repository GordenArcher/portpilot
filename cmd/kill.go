package cmd

import (
	"fmt"
	"strconv"

	"github.com/gordenarcher/portpilot/internal/ports"
	"github.com/spf13/cobra"
)

var forceKill bool

var killCmd = &cobra.Command{
	Use:     "kill <port>",
	Short:   "Kill the process running on a given port",
	Args:    cobra.ExactArgs(1),
	Example: `  portpilot kill 3000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %q must be a number", args[0])
		}

		if err := validatePort(port); err != nil {
			return err
		}

		if !forceKill {
			// Prompt user before killing because this destructive action should never be silent.
			// The force flag skips this prompt for scripting and automation use cases.
			fmt.Printf("Kill process on port %d? [y/N] ", port)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := ports.Kill(port); err != nil {
			return fmt.Errorf("failed to kill process on port %d: %w", port, err)
		}

		fmt.Printf("Killed process on port %d\n", port)
		return nil
	},
}

func init() {
	killCmd.Flags().BoolVarP(&forceKill, "force", "f", false, "Skip confirmation prompt")
}
