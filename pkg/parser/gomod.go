package parser

import (
	"fmt"
	"os"
	"strings"
)

// ParseGoSum parses go.sum files
func ParseGoSum(path string) (map[string]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read go.sum: %w", err)
	}

	deps := make(map[string]Dependency)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			deps[parts[0]+"@"+parts[1]] = Dependency{
				PackageURL:   fmt.Sprintf("pkg:/golang/%s@%s", parts[0], parts[1]),
				Relationship: "direct",
			}
		}
	}

	return deps, nil
}
