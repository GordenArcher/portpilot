package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GordenArcher/portpilot/internal/ports"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	dashboardWidth = 104
	metricWidth    = 21
)

type metric struct {
	Label string
	Value string
	Tone  string
}

var (
	appTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#3B82F6")).
			Padding(0, 2)

	panelStyle = lipgloss.NewStyle().
			Width(dashboardWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3B82F6")).
			Padding(1, 2)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D8491"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB"))
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
	portStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#93C5FD"))

	occupiedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#DC2626")).
			Padding(0, 1)

	reservedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#111827")).
			Background(lipgloss.Color("#FACC15")).
			Padding(0, 1)

	freeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#052E16")).
			Background(lipgloss.Color("#22C55E")).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))

	metricLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Bold(true)
	metricValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB")).Bold(true)
)

// RenderTable prints the scan results as a formatted table.
// reservations is a port to label map used to annotate reserved ports.
func RenderTable(results []ports.PortInfo, reservations map[int]string) {
	rows := make([][]string, 0, len(results)+len(reservations))
	occupiedPorts := make(map[int]bool, len(results))
	reservedOccupied := 0

	for _, r := range results {
		pid := fmt.Sprintf("%d", r.PID)
		if r.PID == 0 {
			pid = dimStyle.Render("unknown")
		}

		label, isReserved := reservations[r.Port]
		status := statusPill("OCCUPIED", "")
		if isReserved {
			status = statusPill("RESERVED", label)
			reservedOccupied++
		}

		rows = append(rows, []string{
			portStyle.Render(fmt.Sprintf("%d", r.Port)),
			pid,
			r.Process,
			r.State,
			status,
		})
		occupiedPorts[r.Port] = true
	}

	// Reserved ports that are currently free still matter because they explain
	// developer intent. Showing them beside live ports makes the scan output a
	// useful workspace map instead of only a raw OS socket listing.
	var unoccupiedReserved []int
	for port := range reservations {
		if !occupiedPorts[port] {
			unoccupiedReserved = append(unoccupiedReserved, port)
		}
	}
	sort.Ints(unoccupiedReserved)

	for _, port := range unoccupiedReserved {
		label := reservations[port]
		rows = append(rows, []string{
			portStyle.Render(fmt.Sprintf("%d", port)),
			dimStyle.Render("unknown"),
			dimStyle.Render("unknown"),
			dimStyle.Render("FREE"),
			statusPill("FREE", label),
		})
	}

	renderDashboard(
		"Port Scan",
		"Live TCP listeners and saved local reservations",
		[]metric{
			{Label: "LISTENING", Value: fmt.Sprintf("%d", len(results)), Tone: "occupied"},
			{Label: "RESERVED", Value: fmt.Sprintf("%d", len(reservations)), Tone: "reserved"},
			{Label: "FREE SAVED", Value: fmt.Sprintf("%d", len(unoccupiedReserved)), Tone: "free"},
			{Label: "MATCHED", Value: fmt.Sprintf("%d", reservedOccupied), Tone: "neutral"},
		},
		[]string{"PORT", "PID", "PROCESS", "STATE", "STATUS"},
		rows,
	)
}

// RenderDetail prints detailed info for a single port (used by `portpilot info`).
func RenderDetail(d *ports.PortDetail) {
	rows := [][]string{
		{"PORT", fmt.Sprintf("%d", d.Port)},
		{"PID", fmt.Sprintf("%d", d.PID)},
		{"PROCESS", d.Process},
		{"STATE", d.State},
	}
	if d.User != "" {
		rows = append(rows, []string{"USER", d.User})
	}
	if d.Command != "" {
		rows = append(rows, []string{"COMMAND", d.Command})
	}

	renderDashboard(
		"Port Detail",
		fmt.Sprintf("Focused process view for port %d", d.Port),
		[]metric{
			{Label: "PORT", Value: fmt.Sprintf("%d", d.Port), Tone: "neutral"},
			{Label: "PID", Value: fmt.Sprintf("%d", d.PID), Tone: "occupied"},
			{Label: "STATE", Value: d.State, Tone: "reserved"},
		},
		[]string{"FIELD", "VALUE"},
		rows,
	)
}

// RenderFreePort shows a successful info result for ports with no listener.
// An unused port is not a command failure. It is often exactly the answer the
// user needs when debugging whether a service can bind safely.
func RenderFreePort(port int) {
	renderDashboard(
		"Port Detail",
		fmt.Sprintf("No listener found on port %d", port),
		[]metric{
			{Label: "PORT", Value: fmt.Sprintf("%d", port), Tone: "neutral"},
			{Label: "STATE", Value: "FREE", Tone: "free"},
			{Label: "PID", Value: "none", Tone: "neutral"},
		},
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"PORT", portStyle.Render(fmt.Sprintf("%d", port))},
			{"STATE", statusPill("FREE", "")},
			{"PID", dimStyle.Render("none")},
			{"PROCESS", dimStyle.Render("none")},
		},
	)
}

// RenderReservations prints all reserved ports as a table.
func RenderReservations(reservations map[int]string) {
	// Sort by port number for deterministic output.
	portNumbers := make([]int, 0, len(reservations))
	for p := range reservations {
		portNumbers = append(portNumbers, p)
	}
	sort.Ints(portNumbers)

	rows := make([][]string, 0, len(portNumbers))
	for _, port := range portNumbers {
		rows = append(rows, []string{
			fmt.Sprintf("%d", port),
			reservations[port],
		})
	}

	renderDashboard(
		"Reserved Ports",
		"Saved labels for ports you care about",
		[]metric{
			{Label: "SAVED LABELS", Value: fmt.Sprintf("%d", len(rows)), Tone: "reserved"},
		},
		[]string{"PORT", "LABEL"},
		rows,
	)
}

// RenderEmpty gives empty states the same visual language as real tables.
// Without this, commands that have no data feel like they failed or skipped
// the UI path, even though an empty result is a valid and useful outcome.
func RenderEmpty(title, message string) {
	fmt.Println()
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		appTitleStyle.Render(title),
		"",
		subtleStyle.Render(message),
	)
	fmt.Println(panelStyle.Render(body))
	fmt.Println()
}

func renderDashboard(title, summary string, metrics []metric, headers []string, rows [][]string) {
	fmt.Println()

	content := []string{
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			appTitleStyle.Render(title),
			"  ",
			subtleStyle.Render(summary),
		),
		"",
		renderMetrics(metrics),
		"",
	}

	if len(rows) == 0 {
		content = append(content, subtleStyle.Render("No rows to show."))
		fmt.Println(panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content...)))
		fmt.Println()
		return
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Width(dashboardWidth - 6).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				return headerStyle.Padding(0, 1)
			case row%2 == 0:
				return lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("#111827"))
			default:
				return lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("#D1D5DB"))
			}
		})

	content = append(content, t.String())
	fmt.Println(panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content...)))
	fmt.Println()
}

func renderMetrics(metrics []metric) string {
	if len(metrics) == 0 {
		return ""
	}

	cards := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		cards = append(cards, renderMetric(metric))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cards...)
}

func renderMetric(m metric) string {
	accent := lipgloss.Color("#60A5FA")
	switch m.Tone {
	case "occupied":
		accent = lipgloss.Color("#F87171")
	case "reserved":
		accent = lipgloss.Color("#FACC15")
	case "free":
		accent = lipgloss.Color("#4ADE80")
	}

	return lipgloss.NewStyle().
		Width(metricWidth).
		MarginRight(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			metricLabelStyle.Render(m.Label),
			metricValueStyle.Render(m.Value),
		))
}

func statusPill(status, label string) string {
	var style lipgloss.Style
	switch status {
	case "OCCUPIED":
		style = occupiedStyle
	case "RESERVED":
		style = reservedStyle
	case "FREE":
		style = freeStyle
	default:
		style = dimStyle
	}

	if strings.TrimSpace(label) == "" {
		return style.Render(status)
	}

	return fmt.Sprintf("%s %s", style.Render(status), subtleStyle.Render(label))
}
