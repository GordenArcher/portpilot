package cmd

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/GordenArcher/portpilot/internal/ports"
	"github.com/spf13/cobra"
)

var forceKill bool
var killAll bool
var killFilterRange string

type killScanFunc func(string) ([]ports.PortInfo, error)
type killPIDFunc func(int) error

var killCmd = &cobra.Command{
	Use:   "kill <port>",
	Short: "Kill the process running on a given port",
	Args:  validateKillArgs,
	Example: `  portpilot kill 3000
  portpilot kill 3000 --force
  portpilot kill --all --filter 3000-9000 --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if killAll {
			return runKillAll(cmd.OutOrStdout(), cmd.InOrStdin(), killFilterRange, forceKill, ports.Scan, ports.KillPID)
		}

		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %q must be a number", args[0])
		}

		if err := validatePort(port); err != nil {
			return err
		}

		if !forceKill {
			ok, err := confirmAction(cmd.OutOrStdout(), cmd.InOrStdin(), fmt.Sprintf("Kill process on port %d? [y/N] ", port))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}
		}

		if err := ports.Kill(port); err != nil {
			if errors.Is(err, ports.ErrPortNotFound) {
				return fmt.Errorf("no process found on port %d", port)
			}
			return fmt.Errorf("failed to kill process on port %d: %w", port, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Killed process on port %d\n", port)
		return nil
	},
}

func init() {
	killCmd.Flags().BoolVarP(&forceKill, "force", "f", false, "Skip confirmation prompt")
	killCmd.Flags().BoolVar(&killAll, "all", false, "Kill every process found by --filter")
	killCmd.Flags().StringVar(&killFilterRange, "filter", "", "Port range for kill --all, for example 3000-9000")
}

func validateKillArgs(cmd *cobra.Command, args []string) error {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	filter, err := cmd.Flags().GetString("filter")
	if err != nil {
		return err
	}

	if all {
		if len(args) != 0 {
			return fmt.Errorf("kill --all does not accept a port argument")
		}
		if strings.TrimSpace(filter) == "" {
			return fmt.Errorf("kill --all requires --filter <range>")
		}
		return nil
	}

	if strings.TrimSpace(filter) != "" {
		return fmt.Errorf("kill --filter requires --all")
	}
	if len(args) != 1 {
		return fmt.Errorf("accepts 1 arg, received %d", len(args))
	}

	return nil
}

// runKillAll implements the destructive range-kill workflow behind
// `portpilot kill --all --filter`.
//
// The command scans first, reduces rows to unique PIDs, prompts unless forced,
// then kills each PID directly. That order matters: prompting after the scan
// lets the user know how many processes will be affected, and deduplicating
// before the prompt keeps the confirmation count aligned with actual signals.
func runKillAll(output io.Writer, input io.Reader, filter string, force bool, scan killScanFunc, killPID killPIDFunc) error {
	results, err := scan(filter)
	if err != nil {
		return fmt.Errorf("scan failed before bulk kill: %w", err)
	}

	targets := uniqueKillTargets(results)
	if len(targets) == 0 {
		fmt.Fprintf(output, "No killable processes found in %s\n", filter)
		return nil
	}

	if !force {
		ok, err := confirmAction(output, input, fmt.Sprintf("Kill %d process(es) in %s? [y/N] ", len(targets), filter))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(output, "Aborted.")
			return nil
		}
	}

	for _, target := range targets {
		if err := killPID(target.PID); err != nil {
			return fmt.Errorf("failed to kill PID %d on port %d: %w", target.PID, target.Port, err)
		}
		fmt.Fprintf(output, "Killed PID %d (%s) from port %d\n", target.PID, target.Process, target.Port)
	}

	return nil
}

type killTarget struct {
	PID     int
	Port    int
	Process string
}

// uniqueKillTargets converts scan rows into process targets for bulk killing.
//
// A single process can listen on multiple ports, and macOS may also report IPv4
// and IPv6 listeners separately. Killing by every row would try to terminate
// the same PID more than once, producing noisy errors after the first signal
// succeeds. Deduplicating by PID gives `kill --all --filter` the behavior users
// expect: every matching process is targeted once.
func uniqueKillTargets(results []ports.PortInfo) []killTarget {
	seen := map[int]bool{}
	targets := make([]killTarget, 0, len(results))

	for _, result := range results {
		if result.PID <= 0 || seen[result.PID] {
			continue
		}

		seen[result.PID] = true
		targets = append(targets, killTarget{
			PID:     result.PID,
			Port:    result.Port,
			Process: result.Process,
		})
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Port == targets[j].Port {
			return targets[i].PID < targets[j].PID
		}
		return targets[i].Port < targets[j].Port
	})

	return targets
}

// confirmAction reads a destructive-action confirmation from the provided
// input stream instead of directly from os.Stdin. That keeps normal CLI prompts
// interactive while making the confirmation path deterministic in tests.
func confirmAction(output io.Writer, input io.Reader, prompt string) (bool, error) {
	fmt.Fprint(output, prompt)

	var confirm string
	if _, err := fmt.Fscanln(input, &confirm); err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	return confirm == "y" || confirm == "Y", nil
}
