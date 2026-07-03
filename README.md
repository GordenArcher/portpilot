# portpilot

`portpilot` is a local developer CLI for seeing and controlling the ports on your machine.

It is meant to make "port already in use" problems obvious: scan what is listening, inspect a specific port, kill the owning process, reserve labels for known services, and watch a port for status changes.

## Commands

```bash
portpilot scan
portpilot scan --filter 3000-9000
portpilot kill 3000
portpilot kill 3000 --force
portpilot info 8080
portpilot reserve 8080 "axon-core"
portpilot reserved
portpilot unreserve 8080
portpilot watch 3000
```

## Example Output

```text
PORT     PID     PROCESS              STATUS
3000     12345   node                 OCCUPIED
8080     67890   axon-core            RESERVED  "axon-core"
5432     11111   postgres             OCCUPIED
9000     unknown unknown              FREE  (reserved: "my-api")
```

## Features

- Scan all occupied TCP listening ports.
- Filter scans to a port range with `--filter 3000-9000`.
- Show detailed process information for one port.
- Kill the process listening on a port, with confirmation by default.
- Skip kill confirmation with `--force` for scripts.
- Reserve a port label such as `axon-core`.
- Remove a port reservation when the label is no longer useful.
- Show reserved ports alongside live scan results.
- Watch a port and print when it becomes free or occupied.

## Project Structure

```text
portpilot/
├── cmd/
│   ├── root.go          # cobra root and command registration
│   ├── scan.go          # portpilot scan
│   ├── kill.go          # portpilot kill
│   ├── info.go          # portpilot info
│   ├── reserve.go       # portpilot reserve
│   ├── reserved.go      # portpilot reserved
│   ├── unreserve.go     # portpilot unreserve
│   └── watch.go         # portpilot watch
├── internal/
│   ├── ports/
│   │   └── scanner.go   # OS-level port scanning, kill, and info helpers
│   ├── store/
│   │   └── reserve.go   # reservations persisted to ~/.portpilot/reservations.json
│   └── ui/
│       └── table.go     # terminal table rendering with lipgloss
├── main.go
├── go.mod
├── go.sum
└── README.md
```

## Tech Stack

| Concern | Package or tool |
| --- | --- |
| CLI framework | `cobra` |
| Terminal styling | `lipgloss` |
| Port scanning on macOS | `lsof` through `os/exec` |
| Port scanning on Linux | `ss` through `os/exec` |
| Reservations persistence | JSON at `~/.portpilot/reservations.json` |
| Watch polling | `net.DialTimeout` plus polling |

## Platform Support

| OS | Scan | Kill | Info | Watch |
| --- | --- | --- | --- | --- |
| macOS | `lsof -iTCP -sTCP:LISTEN -n -P` | `lsof -ti :<port>` plus `kill -9` | `lsof -iTCP :<port> -n -P` | `net.DialTimeout` |
| Linux | `ss -tlnp` | `ss` plus `kill -9` | `ss -tlnp sport = :<port>` | `net.DialTimeout` |

## Build And Install

```bash
go run main.go scan
go build -o portpilot .
go install .
```

## Reservation Storage

Reservations are stored locally at:

```text
~/.portpilot/reservations.json
```

The reservation file is metadata only. Reserving a port does not bind it or prevent another process from using it. It only gives `scan` and `reserved` enough context to show why that port matters.

## Roadmap

- [x] `scan`
- [x] `scan --filter`
- [x] `kill`
- [x] `kill --force`
- [x] `info`
- [x] `reserve`
- [x] `reserved`
- [x] `portpilot unreserve <port>`
- [x] `watch`
- [ ] `portpilot export`
- [ ] Desktop notification when `watch` detects a change
- [ ] GitHub Actions release pipeline
