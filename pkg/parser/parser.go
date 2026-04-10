package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/einride/gh-dependabot/pkg/parser/gomod"
	"github.com/einride/gh-dependabot/pkg/parser/npm"
	"github.com/einride/gh-dependabot/pkg/parser/pip"
	"github.com/einride/gh-dependabot/pkg/parser/python"
	"github.com/einride/gh-dependabot/pkg/parser/types"
)

// ParseLockfile detects and parses a lockfile based on its name
func ParseLockfile(path string) (map[string]types.Dependency, error) {
	base := strings.ToLower(filepath.Base(path))

	switch base {
	case "package-lock.json", "package.json":
		return npm.ParsePackageLock(path)
	case "requirements.txt":
		return pip.ParseRequirementsTxt(path)
	case "uv.lock":
		return python.ParseUVLock(path)
	case "go.sum":
		return gomod.ParseGoSum(path)
	case "yarn.lock", "pnpm-lock.yaml":
		return npm.ParseYarnLock(path)
	default:
		return nil, fmt.Errorf("unsupported lockfile format: %s", filepath.Base(path))
	}
}
