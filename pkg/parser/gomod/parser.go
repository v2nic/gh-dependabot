package gomod

import (
	"fmt"
	"os"
	"strings"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

// ParseGoSum parses go.sum files
func ParseGoSum(path string) (map[string]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read go.sum: %w", err)
	}

	deps := make(map[string]types.Dependency)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			deps[parts[0]+"@"+parts[1]] = types.Dependency{
				PackageURL:   fmt.Sprintf("pkg:/golang/%s@%s", parts[0], parts[1]),
				Relationship: "direct",
			}
		}
	}

	return deps, nil
}
