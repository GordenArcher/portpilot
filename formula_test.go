package main

import (
	"os"
	"strings"
	"testing"
)

func TestHomebrewFormulaMatchesCurrentRelease(t *testing.T) {
	data, err := os.ReadFile("Formula/portpilot.rb")
	if err != nil {
		t.Fatalf("read Homebrew formula: %v", err)
	}

	formula := string(data)
	requiredSnippets := []string{
		`class Portpilot < Formula`,
		`homepage "https://github.com/GordenArcher/portpilot"`,
		`url "https://github.com/GordenArcher/portpilot/archive/refs/tags/v0.1.1.tar.gz"`,
		`sha256 "0b4dba35d2f49dc8745eb77534f23b4a930632b844725247830f0147e0899367"`,
		`depends_on "go" => :build`,
		`system "go", "build", "-trimpath", "-ldflags=-s -w", "-o", bin/"portpilot", "."`,
		`assert_match "portpilot lets you scan", shell_output("#{bin}/portpilot --help")`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(formula, snippet) {
			t.Fatalf("Homebrew formula missing required snippet %q\n%s", snippet, formula)
		}
	}
}
