package notify

import (
	"runtime"
	"strings"
	"testing"
)

func TestEscapeAppleScriptEscapesQuotesAndBackslashes(t *testing.T) {
	got := escapeAppleScript(`Port "7000" at C:\tmp`)
	want := `Port \"7000\" at C:\\tmp`
	if got != want {
		t.Fatalf("escape mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestSendUsesOsascriptOnDarwinAndNoopsElsewhere(t *testing.T) {
	originalRunCommand := runCommand
	defer func() {
		runCommand = originalRunCommand
	}()

	var called bool
	var gotName string
	var gotArgs []string
	runCommand = func(name string, args ...string) error {
		called = true
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := Send("portpilot", `Port "7000" is now FREE`); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if runtime.GOOS != "darwin" {
		if called {
			t.Fatalf("non darwin platforms should not execute notification command")
		}
		return
	}

	if !called {
		t.Fatalf("darwin notification should execute osascript")
	}
	if gotName != "osascript" {
		t.Fatalf("command name mismatch: %q", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-e" {
		t.Fatalf("osascript args mismatch: %#v", gotArgs)
	}
	if !strings.Contains(gotArgs[1], `display notification "Port \"7000\" is now FREE" with title "portpilot"`) {
		t.Fatalf("osascript payload mismatch: %q", gotArgs[1])
	}
}
