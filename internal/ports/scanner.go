package ports

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	minPort = 1
	maxPort = 65535
)

// PortInfo holds everything we know about a port at scan time.
type PortInfo struct {
	Port    int
	PID     int
	Process string
	State   string // "LISTEN", "ESTABLISHED", etc.
}

// PortDetail is the richer version used by the info command.
type PortDetail struct {
	PortInfo
	User    string
	Command string
}

// Scan returns all occupied ports on this machine, optionally filtered by
// a range string like "3000-9000". An empty filterRange means all ports.
func Scan(filterRange string) ([]PortInfo, error) {
	low, high, err := parseRange(filterRange)
	if err != nil {
		return nil, err
	}

	switch runtime.GOOS {
	case "darwin":
		return scanDarwin(low, high)
	case "linux":
		return scanLinux(low, high)
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// IsOccupied returns true if anything is listening on the given port.
// Used by the watch command to poll for status changes.
func IsOccupied(port int) (bool, error) {
	if err := validatePort(port); err != nil {
		return false, err
	}

	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return false, nil
	}
	conn.Close()
	return true, nil
}

// Info returns detailed information about the process using a port.
func Info(port int) (*PortDetail, error) {
	if err := validatePort(port); err != nil {
		return nil, err
	}

	switch runtime.GOOS {
	case "darwin":
		return infoDarwin(port)
	case "linux":
		return infoLinux(port)
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// Kill sends SIGKILL to the process occupying the given port.
func Kill(port int) error {
	if err := validatePort(port); err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return killDarwin(port)
	case "linux":
		return killLinux(port)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// scanDarwin uses lsof to list listening ports on macOS.
func scanDarwin(low, high int) ([]PortInfo, error) {
	// These lsof arguments list only listening TCP ports, skip hostname lookup,
	// and skip port name resolution so the output stays fast and predictable.
	out, err := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P").Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	return parseLsofOutput(string(out), low, high)
}

// scanLinux uses ss (socket stats) which ships with all modern Linux distros.
func scanLinux(low, high int) ([]PortInfo, error) {
	// These ss arguments list listening TCP ports numerically and include process details.
	out, err := exec.Command("ss", "-tlnp").Output()
	if err != nil {
		return nil, fmt.Errorf("ss failed: %w", err)
	}

	return parseSsOutput(string(out), low, high)
}

// parseLsofOutput parses lsof output into PortInfo slice.
// Example line:
// node    12345 user   23u  IPv4 0x...  0t0  TCP *:3000 (LISTEN)
func parseLsofOutput(raw string, low, high int) ([]PortInfo, error) {
	var results []PortInfo
	portRe := regexp.MustCompile(`:(\d+) \(LISTEN\)`)

	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		process := fields[0]
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		match := portRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		port, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}

		if !inRange(port, low, high) {
			continue
		}

		results = append(results, PortInfo{
			Port:    port,
			PID:     pid,
			Process: process,
			State:   "LISTEN",
		})
	}

	return results, nil
}

// parseSsOutput parses ss output into PortInfo slice.
// Example line:
// LISTEN 0 128 0.0.0.0:8080 0.0.0.0:* users:(("axon core",pid=9876,fd=7))
func parseSsOutput(raw string, low, high int) ([]PortInfo, error) {
	var results []PortInfo
	portRe := regexp.MustCompile(`:(\d+)\s`)
	pidRe := regexp.MustCompile(`pid=(\d+)`)
	procRe := regexp.MustCompile(`\(\("([^"]+)"`)

	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, "LISTEN") {
			continue
		}

		portMatch := portRe.FindStringSubmatch(line)
		if portMatch == nil {
			continue
		}

		port, err := strconv.Atoi(portMatch[1])
		if err != nil {
			continue
		}

		if !inRange(port, low, high) {
			continue
		}

		pid := 0
		process := "unknown"

		if m := pidRe.FindStringSubmatch(line); m != nil {
			pid, _ = strconv.Atoi(m[1])
		}
		if m := procRe.FindStringSubmatch(line); m != nil {
			process = m[1]
		}

		results = append(results, PortInfo{
			Port:    port,
			PID:     pid,
			Process: process,
			State:   "LISTEN",
		})
	}

	return results, nil
}

func infoDarwin(port int) (*PortDetail, error) {
	out, err := exec.Command("lsof", "-iTCP", fmt.Sprintf(":%d", port), "-n", "-P").Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("nothing found on port %d", port)
	}

	// First line is the header, second is the actual entry.
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return nil, fmt.Errorf("unexpected lsof output format")
	}

	pid, _ := strconv.Atoi(fields[1])
	return &PortDetail{
		PortInfo: PortInfo{
			Port:    port,
			PID:     pid,
			Process: fields[0],
			State:   "LISTEN",
		},
		User:    fields[2],
		Command: fields[0],
	}, nil
}

func infoLinux(port int) (*PortDetail, error) {
	out, err := exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%d", port)).Output()
	if err != nil {
		return nil, fmt.Errorf("ss failed: %w", err)
	}

	results, err := parseSsOutput(string(out), 0, 65535)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("nothing found on port %d", port)
	}

	return &PortDetail{
		PortInfo: results[0],
		Command:  results[0].Process,
	}, nil
}

func killDarwin(port int) error {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("no process found on port %d", port)
	}

	pid := strings.TrimSpace(string(out))
	return exec.Command("kill", "-9", pid).Run()
}

func killLinux(port int) error {
	results, err := scanLinux(port, port)
	if err != nil || len(results) == 0 {
		return fmt.Errorf("no process found on port %d", port)
	}

	pid := strconv.Itoa(results[0].PID)
	return exec.Command("kill", "-9", pid).Run()
}

func parseRange(r string) (int, int, error) {
	if r == "" {
		return minPort, maxPort, nil
	}

	parts := strings.SplitN(r, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range %q expected format 3000-9000", r)
	}

	low, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid low port in range: %q", parts[0])
	}

	high, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid high port in range: %q", parts[1])
	}

	if err := validatePort(low); err != nil {
		return 0, 0, err
	}
	if err := validatePort(high); err != nil {
		return 0, 0, err
	}
	if low > high {
		return 0, 0, fmt.Errorf("invalid range %q low port cannot be greater than high port", r)
	}

	return low, high, nil
}

func inRange(port, low, high int) bool {
	return port >= low && port <= high
}

func validatePort(port int) error {
	if port < minPort || port > maxPort {
		return fmt.Errorf("invalid port %d must be between %d and %d", port, minPort, maxPort)
	}

	return nil
}
