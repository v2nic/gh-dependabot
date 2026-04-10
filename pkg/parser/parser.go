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
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	switch {
	case base == "package-lock.json" || ext == ".json" && strings.Contains(base, "package"):
		return parsePackageLock(path)
	case base == "requirements.txt":
		return parseRequirementsTxt(path)
	case base == "uv.lock":
		return parseUVLock(path)
	case base == "go.sum":
		return parseGoSum(path)
	case base == "yarn.lock" || base == "pnpm-lock.yaml":
		return parseNPMWorkspaces(path)
	default:
		return parsePackageLock(path) // fallback to npm
	}
}

func parsePackageLock(path string) (map[string]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var lockfile struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		Dependencies map[string]map[string]interface{} `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &lockfile); err != nil {
		// Try alternative format (package.json style)
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

		// Extract package name from node_modules path
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

		// Clean version string
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

		// Parse package==version or package>=version
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
	// Handle ==, >=, <=, ~, >
	for _, sep := range []string{"==", ">=", "<=", "~=", ">", "<", " "} {
		if idx := strings.Index(line, sep); idx > 0 {
			name = strings.TrimSpace(line[:idx])
			version = strings.TrimSpace(line[idx+len(sep):])
			// Remove trailing comments and extras
			if idx := strings.Index(version, "#"); idx >= 0 {
				version = strings.TrimSpace(version[:idx])
			}
			if idx := strings.Index(version, ";"); idx >= 0 {
				version = strings.TrimSpace(version[:idx])
			}
			return
		}
	}
	return line, "latest"
}

func parseUVLock(path string) (map[string]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read uv.lock: %w", err)
	}

	deps := make(map[string]Dependency)

	// Parse TOML-like format
	// Format is typically: package-name = { version = "x.y.z", ... }
	lines := strings.Split(string(data), "\n")
	var currentPkg string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for package name header [package.name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentPkg = strings.Trim(line, "[]")
			continue
		}

		// Check for version = "x.y.z"
		if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				version := strings.Trim(strings.Trim(parts[1], ` "`), " ")
				if version != "" && currentPkg != "" {
					deps[currentPkg+"@"+version] = Dependency{
						PackageURL:   fmt.Sprintf("pkg:/python/%s@%s", currentPkg, version),
						Relationship: "direct",
					}
				}
			}
		}
	}

	// If no structured format found, try JSON format
	if len(deps) == 0 {
		var lockfile struct {
			Packages []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"packages"`
		}
		if err := json.Unmarshal(data, &lockfile); err == nil {
			for _, pkg := range lockfile.Packages {
				if pkg.Name != "" && pkg.Version != "" {
					deps[pkg.Name+"@"+pkg.Version] = Dependency{
						PackageURL:   fmt.Sprintf("pkg:/python/%s@%s", pkg.Name, pkg.Version),
						Relationship: "direct",
					}
				}
			}
		}
	}

	return deps, nil
}

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

		// go.sum format: module version hash
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			version := parts[1]

			deps[name+"@"+version] = Dependency{
				PackageURL:   fmt.Sprintf("pkg:/golang/%s@%s", name, version),
				Relationship: "direct",
			}
		}
	}

	return deps, nil
}

func parseNPMWorkspaces(path string) (map[string]Dependency, error) {
	// For yarn.lock and pnpm-lock.yaml, try to extract dependencies
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	deps := make(map[string]Dependency)

	// Simple extraction - find "package@version" patterns
	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Look for patterns like: "lodash@^4.17.21"
		if strings.Contains(line, "@") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Extract package references
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

func splitAtVersion(s string) (name, version string) {
	// Handle scoped packages like @types/node
	atCount := strings.Count(s, "@")
	if atCount == 1 {
		parts := strings.SplitN(s, "@", 2)
		return parts[0], strings.Trim(parts[1], " \t\n\r")
	} else if atCount >= 2 {
		// Scoped: @scope/package@version
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
	// node_modules/pkg or @scope/pkg
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
	// Remove leading 'v' or 'x.y.z-beta+build' style versions
	version = strings.TrimPrefix(version, "v")
	// Keep only the semantic version part
	if idx := strings.Index(version, "-"); idx > 0 {
		version = version[:idx]
	}
	return strings.TrimSpace(version)
}