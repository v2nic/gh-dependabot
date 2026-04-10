package parser

import (
	"fmt"
	"os"
	"strings"
)

// ParseRequirementsTxt parses Python requirements.txt files
func ParseRequirementsTxt(path string) (map[string]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read requirements.txt: %w", err)
	}

	deps := make(map[string]Dependency)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		name, version := parseRequirementLine(line)
		if name == "" {
			continue
		}

		deps[name+"@"+version] = Dependency{
			PackageURL:   fmt.Sprintf("pkg:/pypi/%s@%s", name, version),
			Relationship: "direct",
		}
	}

	return deps, nil
}

func parseRequirementLine(line string) (name, version string) {
	for _, sep := range []string{"==", ">=", "<=", "~=", ">", "<", " "} {
		if idx := strings.Index(line, sep); idx > 0 {
			name = strings.TrimSpace(line[:idx])
			version = strings.TrimSpace(line[idx+len(sep):])
			version = strings.TrimSuffix(version, "#")
			version = strings.TrimSuffix(version, ";")
			return name, strings.TrimSpace(version)
		}
	}
	return line, "latest"
}
