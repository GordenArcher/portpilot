package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type commandRunner func(name string, args ...string) error

var runCommand commandRunner = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// Send displays a desktop notification when the current OS supports it.
//
// macOS uses osascript because it is available without adding a dependency or
// requiring a background daemon. Other platforms are intentionally no-ops for
// now, which keeps watch useful everywhere while preserving a clear extension
// point for Linux notification providers later.
func Send(title, message string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	script := fmt.Sprintf(
		`display notification "%s" with title "%s"`,
		escapeAppleScript(message),
		escapeAppleScript(title),
	)

	if err := runCommand("osascript", "-e", script); err != nil {
		return fmt.Errorf("send desktop notification: %w", err)
	}

	return nil
}

func escapeAppleScript(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
