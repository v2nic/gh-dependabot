package parser

import (
	"path/filepath"
	"strings"
)

// ParseLockfile detects and parses a lockfile based on its name
func ParseLockfile(path string) (map[string]Dependency, error) {
	base := strings.ToLower(filepath.Base(path))

	switch base {
	case "package-lock.json", "package.json":
		return ParsePackageLock(path)
	case "requirements.txt":
		return ParseRequirementsTxt(path)
	case "uv.lock":
		return ParseUVLock(path)
	case "go.sum":
		return ParseGoSum(path)
	case "yarn.lock", "pnpm-lock.yaml":
		return ParseYarnLock(path)
	default:
		return ParsePackageLock(path)
	}
}
