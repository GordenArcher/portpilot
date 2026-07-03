package cmd

import "fmt"

const (
	minPort = 1
	maxPort = 65535
)

func validatePort(port int) error {
	if port < minPort || port > maxPort {
		return fmt.Errorf("invalid port %d must be between %d and %d", port, minPort, maxPort)
	}

	return nil
}
