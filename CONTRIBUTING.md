# Contributing to portpilot

Thanks for helping improve `portpilot`. This project is a small CLI, but it touches local processes, ports, and kill signals, so changes need to be careful, testable, and easy to review.

## Development Setup

Install Go 1.22 or newer.

Clone the repository:

```bash
git clone https://github.com/GordenArcher/portpilot.git
cd portpilot
```

Run the full test suite:

```bash
go test ./...
```

Run lint locally:

```bash
golangci-lint run ./...
```

Build the CLI:

```bash
go build -o portpilot .
```

Run a local command:

```bash
go run . scan
```

## Contribution Workflow

Keep changes focused. A good pull request should do one clear thing, such as adding a command flag, fixing scanner behavior, improving terminal output, or updating release automation.

Before opening a pull request, run:

```bash
go test ./...
golangci-lint run ./...
```

Normal pushes to `main` and pull requests run the same checks in GitHub Actions.

## Testing Expectations

Add tests for behavior, not just implementation details.

For command changes, prefer extracting the core flow into a helper that accepts injected functions or writers. That lets tests verify behavior without depending on real local ports, real process killing, or desktop notification permissions.

For storage changes, use `t.TempDir()` and `t.Setenv("HOME", dir)` so tests never touch a contributor's real `~/.portpilot` data.

For output modes intended for scripts, keep stdout machine readable. For example, JSON output must not include loading spinners, dashboard borders, or warning text.

## Linting Rules

The project uses `golangci-lint` with `funlen` and `nolintlint` enabled.

Do not silence `funlen` casually. If a function is too long, split it into smaller units that make the flow easier to test and review.

If a `nolint` directive is ever necessary, it must be specific and include an explanation. Unused `nolint` directives fail lint.

## Local Safety

Be careful with commands that kill processes.

Tests should not call real process killing paths unless they are explicitly isolated and safe. Prefer testing command behavior with injected kill functions, then keep the actual OS signal primitive small.

Manual smoke tests for destructive commands should use known disposable processes.

## Release Process

Releases are tag based.

Create and push a version tag:

```bash
git tag v0.1.2
git push origin v0.1.2
```

The release workflow runs tests, builds the release binaries, and uploads:

```text
portpilot-darwin-amd64
portpilot-darwin-arm64
portpilot-linux-amd64
portpilot-linux-arm64
```

After a release, update the Homebrew formula URL and sha256 if the formula should point at the new version.

## Roadmap Work

Roadmap work should be completed one item at a time.

For each item:

1. Implement the behavior.
2. Add focused tests.
3. Run `go test ./...`.
4. Run `golangci-lint run ./...`.
5. Commit the item separately.

Demo media for the README should be added separately from feature work so the media can be reviewed on its own.
