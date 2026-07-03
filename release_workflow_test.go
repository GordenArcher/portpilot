package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseWorkflowCoversPlannedReleaseContract(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(data)
	requiredSnippets := []string{
		`tags:`,
		`"v*"`,
		`contents: write`,
		`actions/checkout@v4`,
		`actions/setup-go@v5`,
		`go-version: "1.22.x"`,
		`go test ./...`,
		`softprops/action-gh-release@v2`,
		`generate_release_notes: true`,
		`files: dist/*`,
		`portpilot-${goos}-${goarch}`,
		`darwin amd64`,
		`darwin arm64`,
		`linux amd64`,
		`linux arm64`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(workflow, snippet) {
			t.Fatalf("release workflow missing required snippet %q\n%s", snippet, workflow)
		}
	}
}
