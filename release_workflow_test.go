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

func TestCIWorkflowRunsTestsOnNormalPushes(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}

	workflow := string(data)
	requiredSnippets := []string{
		`push:`,
		`branches:`,
		`main`,
		`pull_request:`,
		`contents: read`,
		`actions/checkout@v4`,
		`actions/setup-go@v5`,
		`go-version: "1.22.x"`,
		`go test ./...`,
		`golangci/golangci-lint-action@v8`,
		`version: v2.12.2`,
		`args: ./...`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(workflow, snippet) {
			t.Fatalf("ci workflow missing required snippet %q\n%s", snippet, workflow)
		}
	}
}

func TestGolangCILintConfigEnforcesFunlenAndNolintlint(t *testing.T) {
	data, err := os.ReadFile(".golangci.yaml")
	if err != nil {
		t.Fatalf("read golangci config: %v", err)
	}

	config := string(data)
	requiredSnippets := []string{
		`funlen`,
		`nolintlint`,
		`lines: 80`,
		`statements: 45`,
		`require-explanation: true`,
		`require-specific: true`,
		`allow-unused: false`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(config, snippet) {
			t.Fatalf("golangci config missing required snippet %q\n%s", snippet, config)
		}
	}
}
