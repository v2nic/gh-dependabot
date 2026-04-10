package python

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

// ParseUVLock parses uv.lock files (Python)
func ParseUVLock(path string) (map[string]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read uv.lock: %w", err)
	}

	// Try JSON format first
	var lockfile struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lockfile); err == nil && len(lockfile.Packages) > 0 {
		deps := make(map[string]types.Dependency)
		for _, pkg := range lockfile.Packages {
			if pkg.Name != "" && pkg.Version != "" {
				deps[pkg.Name+"@"+pkg.Version] = types.Dependency{
					PackageURL:   fmt.Sprintf("pkg:/python/%s@%s", pkg.Name, pkg.Version),
					Relationship: "direct",
				}
			}
		}
		return deps, nil
	}

	// Fallback to TOML-like format
	deps := make(map[string]types.Dependency)
	lines := strings.Split(string(data), "\n")
	var currentPkg string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentPkg = strings.Trim(line, "[]")
			continue
		}

		if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				version := strings.Trim(parts[1], `" `)
				if version != "" && currentPkg != "" {
					deps[currentPkg+"@"+version] = types.Dependency{
						PackageURL:   fmt.Sprintf("pkg:/python/%s@%s", currentPkg, version),
						Relationship: "direct",
					}
				}
			}
		}
	}

	return deps, nil
}
