package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Dependency struct {
	PackageURL   string `json:"package_url"`
	Relationship string `json:"relationship"`
	Scope        string `json:"scope,omitempty"`
}

// ParseLockfile detects and parses a lockfile, returning package-urls for the dependency graph API
func ParseLockfile(path string) (map[string]Dependency, error) {
	base := strings.ToLower(filepath.Base(path))

	switch {
	case base == "package-lock.json" || base == "package.json":
		return parsePackageLock(path)
	case base == "requirements.txt":
		return parseRequirementsTxt(path)
	case base == "uv.lock":
		return parseUVLock(path)
	case base == "go.sum":
		return parseGoSum(path)
	case base == "yarn.lock", base == "pnpm-lock.yaml":
		return parseYarnLock(path)
	default:
		return parsePackageLock(path)
	}
}

// =============================================================================
// NPM (package-lock.json)
// =============================================================================

func parsePackageLock(path string) (map[string]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var lockfile struct {
		Packages    map[string]struct{ Version string } `json:"packages"`
		Dependencies map[string]map[string]interface{} `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &lockfile); err != nil {
		var pkgs []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err2 := json.Unmarshal(data, &pkgs); err2 == nil {
			return parsePackageJSON(pkgs), nil
		}
		return nil, fmt.Errorf("parse package-lock.json: %w", err)
	}

	deps := make(map[string]Dependency)
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
		deps[pkgName+"@"+version] = Dependency{
			PackageURL:   fmt.Sprintf("pkg:/npm/%s@%s", pkgName, version),
			Relationship: "direct",
			Scope:        "runtime",
		}
	}

	return deps, nil
}

func parsePackageJSON(pkgs []struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}) map[string]Dependency {
	deps := make(map[string]Dependency)
	for _, pkg := range pkgs {
		if pkg.Name == "" || pkg.Version == "" {
			continue
		}
		deps[pkg.Name+"@"+pkg.Version] = Dependency{
			PackageURL:   fmt.Sprintf("pkg:/npm/%s@%s", pkg.Name, pkg.Version),
			Relationship: "direct",
		}
	}
	return deps
}

func parseYarnLock(path string) (map[string]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read yarn.lock: %w", err)
	}

	deps := make(map[string]Dependency)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.Contains(line, "@") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			words := strings.Fields(line)
			for _, word := range words {
				if strings.Contains(word, "@") && !strings.HasPrefix(word, "@@") {
					if pkg, ver := splitAtVersion(word); pkg != "" && ver != "" {
						deps[pkg+"@"+ver] = Dependency{
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
// Python (requirements.txt, uv.lock)
// =============================================================================

func parseRequirementsTxt(path string) (map[string]Dependency, error) {
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

func parseUVLock(path string) (map[string]Dependency, error) {
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
		deps := make(map[string]Dependency)
		for _, pkg := range lockfile.Packages {
			if pkg.Name != "" && pkg.Version != "" {
				deps[pkg.Name+"@"+pkg.Version] = Dependency{
					PackageURL:   fmt.Sprintf("pkg:/python/%s@%s", pkg.Name, pkg.Version),
					Relationship: "direct",
				}
			}
		}
		return deps, nil
	}

	// Fallback to TOML-like format
	deps := make(map[string]Dependency)
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
					deps[currentPkg+"@"+version] = Dependency{
						PackageURL:   fmt.Sprintf("pkg:/python/%s@%s", currentPkg, version),
						Relationship: "direct",
					}
				}
			}
		}
	}

	return deps, nil
}

// =============================================================================
// Go (go.sum)
// =============================================================================

func parseGoSum(path string) (map[string]Dependency, error) {
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
	parts := strings.SplitN(path, "/", 1)
	return parts[0]
}

func cleanVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	if idx := strings.Index(version, "-"); idx > 0 {
		version = version[:idx]
	}
	return strings.TrimSpace(version)
}