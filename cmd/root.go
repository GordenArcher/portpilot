package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portpilot",
	Short: "portpilot gives developers port visibility and control",
	Long: `portpilot lets you scan, kill, reserve, and watch ports on your machine.
Use it to stop fighting "port already in use" errors and know exactly what's running where.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(reserveCmd)
	rootCmd.AddCommand(reservedCmd)
	rootCmd.AddCommand(watchCmd)
}
