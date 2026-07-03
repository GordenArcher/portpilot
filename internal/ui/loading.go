package ui

import (
	"fmt"
	"os"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const minimumLoadingTime = 650 * time.Millisecond

// WithLoading renders a compact loading indicator while a command does work.
//
// The spinner only animates for interactive terminals. When stdout is being
// captured by a script or test, it prints one plain line instead. That keeps
// portpilot pleasant for humans without leaking carriage returns and ANSI
// clearing sequences into machine readable logs.
func WithLoading(message string, work func() error) error {
	started := time.Now()

	if !isTerminal() {
		fmt.Println(loadingLine(message, 0))
		err := work()
		if remaining := minimumLoadingTime - time.Since(started); remaining > 0 {
			time.Sleep(remaining)
		}
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- work()
	}()

	ticker := time.NewTicker(90 * time.Millisecond)
	defer ticker.Stop()

	var result error
	frame := 0
	workDone := false
	for {
		select {
		case err := <-done:
			result = err
			workDone = true
			if time.Since(started) >= minimumLoadingTime {
				clearLoadingLine()
				return result
			}
		case <-ticker.C:
			if workDone && time.Since(started) >= minimumLoadingTime {
				clearLoadingLine()
				return result
			}
			fmt.Printf("\r\033[2K%s", loadingLine(message, frame))
			frame++
		}
	}
}

func loadingLine(message string, frame int) string {
	return fmt.Sprintf(
		"%s %s",
		borderStyle.Render(spinnerFrames[frame%len(spinnerFrames)]),
		subtleStyle.Render(message),
	)
}

func clearLoadingLine() {
	fmt.Print("\r\033[2K")
}

func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
