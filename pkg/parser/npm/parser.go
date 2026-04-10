package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

// ParsePackageLock parses npm package-lock.json or package.json files
func ParsePackageLock(path string) (map[string]types.Dependency, error) {
	base := strings.ToLower(filepath.Base(path))

	if base == "package.json" {
		return parsePackageJSON(path)
	}
	return parsePackageLockJSON(path)
}

func parsePackageLockJSON(path string) (map[string]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var lockfile struct {
		Packages     map[string]struct{ Version string } `json:"packages"`
		Dependencies map[string]map[string]interface{}   `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, fmt.Errorf("parse package-lock.json: %w", err)
	}

	deps := make(map[string]types.Dependency)
	for name, pkg := range lockfile.Packages {
		if name == "" || name == "node_modules" {
			continue
		}

		pkgName := extractPackageName(name)
		if pkgName == "" {
			continue
		}

		version := pkg.Version
		if version == "" {
			if dep, ok := lockfile.Dependencies[pkgName]; ok {
				if v, ok := dep["version"].(string); ok {
					version = v
				}
			}
		}

		version = cleanVersion(version)
		deps[pkgName+"@"+version] = types.Dependency{
			PackageURL:   fmt.Sprintf("pkg:/npm/%s@%s", pkgName, version),
			Relationship: "direct",
			Scope:        "runtime",
		}
	}

	return deps, nil
}

func parsePackageJSON(path string) (map[string]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}

	var pkgs []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	deps := make(map[string]types.Dependency)
	for _, pkg := range pkgs {
		if pkg.Name == "" || pkg.Version == "" {
			continue
		}
		deps[pkg.Name+"@"+pkg.Version] = types.Dependency{
			PackageURL:   fmt.Sprintf("pkg:/npm/%s@%s", pkg.Name, pkg.Version),
			Relationship: "direct",
		}
	}
	return deps, nil
}

// ParseYarnLock parses yarn.lock and pnpm-lock.yaml files
func ParseYarnLock(path string) (map[string]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read yarn.lock: %w", err)
	}

	deps := make(map[string]types.Dependency)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.Contains(line, "@") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			words := strings.Fields(line)
			for _, word := range words {
				if strings.Contains(word, "@") && !strings.HasPrefix(word, "@@") {
					if pkg, ver := splitAtVersion(word); pkg != "" && ver != "" {
						deps[pkg+"@"+ver] = types.Dependency{
							PackageURL:   fmt.Sprintf("pkg:/npm/%s@%s", pkg, ver),
							Relationship: "direct",
						}
					}
				}
			}
		}
	}

	return deps, nil
}

// =============================================================================
// Utility functions
// =============================================================================

func splitAtVersion(s string) (name, version string) {
	atCount := strings.Count(s, "@")
	if atCount == 1 {
		parts := strings.SplitN(s, "@", 2)
		return parts[0], strings.Trim(parts[1], " \t\n\r")
	}
	if atCount >= 2 {
		firstAt := strings.Index(s, "@")
		secondAt := strings.Index(s[firstAt+1:], "@")
		if secondAt > 0 {
			name = s[:firstAt+1+secondAt]
			version = s[firstAt+1+secondAt+1:]
			return name, version
		}
	}
	return "", ""
}

func extractPackageName(nodeModulesPath string) string {
	path := strings.TrimPrefix(nodeModulesPath, "node_modules/")
	if strings.HasPrefix(path, "@") {
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	parts := strings.SplitN(path, "/", 2)
	return parts[0]
}

func cleanVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	if idx := strings.Index(version, "-"); idx > 0 {
		version = version[:idx]
	}
	return strings.TrimSpace(version)
}
