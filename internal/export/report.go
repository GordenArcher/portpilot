package export

import (
	"encoding/json"
	"io"
	"sort"
	"time"

	"github.com/gordenarcher/portpilot/internal/ports"
)

type Summary struct {
	Listening           int `json:"listening"`
	Reserved            int `json:"reserved"`
	FreeReservations    int `json:"free_reservations"`
	MatchedReservations int `json:"matched_reservations"`
}

type PortRecord struct {
	Port        int    `json:"port"`
	PID         int    `json:"pid,omitempty"`
	Process     string `json:"process,omitempty"`
	State       string `json:"state"`
	Status      string `json:"status"`
	Reservation string `json:"reservation,omitempty"`
}

type Report struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Filter      string       `json:"filter,omitempty"`
	Summary     Summary      `json:"summary"`
	Ports       []PortRecord `json:"ports"`
}

// NewReport converts live scan results plus saved reservations into the stable
// JSON shape used by `portpilot export`.
//
// I keep this transformation outside the Cobra command because export is a
// contract. Tests should be able to exercise ordering, summary counts, and
// reserved free ports without needing whatever happens to be listening on the
// developer's machine at that moment.
func NewReport(now time.Time, filter string, results []ports.PortInfo, reservations map[int]string) Report {
	records := make([]PortRecord, 0, len(results)+len(reservations))
	occupiedPorts := make(map[int]bool, len(results))
	matchedReservations := 0

	sortedResults := append([]ports.PortInfo(nil), results...)
	sort.SliceStable(sortedResults, func(i, j int) bool {
		if sortedResults[i].Port == sortedResults[j].Port {
			return sortedResults[i].PID < sortedResults[j].PID
		}
		return sortedResults[i].Port < sortedResults[j].Port
	})

	for _, result := range sortedResults {
		record := PortRecord{
			Port:    result.Port,
			PID:     result.PID,
			Process: result.Process,
			State:   result.State,
			Status:  "occupied",
		}

		if label, ok := reservations[result.Port]; ok {
			record.Status = "reserved"
			record.Reservation = label
			matchedReservations++
		}

		records = append(records, record)
		occupiedPorts[result.Port] = true
	}

	freeReservedPorts := make([]int, 0, len(reservations))
	for port := range reservations {
		if !occupiedPorts[port] {
			freeReservedPorts = append(freeReservedPorts, port)
		}
	}
	sort.Ints(freeReservedPorts)

	for _, port := range freeReservedPorts {
		records = append(records, PortRecord{
			Port:        port,
			State:       "FREE",
			Status:      "free_reserved",
			Reservation: reservations[port],
		})
	}

	return Report{
		GeneratedAt: now.UTC(),
		Filter:      filter,
		Summary: Summary{
			Listening:           len(results),
			Reserved:            len(reservations),
			FreeReservations:    len(freeReservedPorts),
			MatchedReservations: matchedReservations,
		},
		Ports: records,
	}
}

func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
