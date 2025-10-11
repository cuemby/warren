package dns

import (
	"fmt"
	"strconv"
	"strings"
)

// parseInstanceName parses an instance-specific DNS name
//
// Supports formats:
//   - nginx-1 -> serviceName="nginx", instance=1
//   - nginx-2 -> serviceName="nginx", instance=2
//   - web-api-3 -> serviceName="web-api", instance=3
//
// Returns:
//   - serviceName: the service name
//   - instanceNum: the instance number (1-indexed)
//   - error: if the name doesn't match the instance pattern
func parseInstanceName(name string) (serviceName string, instanceNum int, err error) {
	// Instance names follow the pattern: <service-name>-<number>
	// Examples: nginx-1, nginx-2, web-api-3

	// Find the last hyphen
	lastHyphen := strings.LastIndex(name, "-")
	if lastHyphen == -1 {
		return "", 0, fmt.Errorf("not an instance name (no hyphen): %s", name)
	}

	// Extract potential service name and instance number
	potentialService := name[:lastHyphen]
	potentialNumber := name[lastHyphen+1:]

	// Try to parse the number
	num, err := strconv.Atoi(potentialNumber)
	if err != nil {
		return "", 0, fmt.Errorf("not an instance name (invalid number): %s", name)
	}

	// Instance numbers must be positive
	if num < 1 {
		return "", 0, fmt.Errorf("instance number must be >= 1: %s", name)
	}

	return potentialService, num, nil
}

// makeInstanceName creates an instance-specific DNS name
// Example: makeInstanceName("nginx", 1) -> "nginx-1"
func makeInstanceName(serviceName string, instanceNum int) string {
	return fmt.Sprintf("%s-%d", serviceName, instanceNum)
}
