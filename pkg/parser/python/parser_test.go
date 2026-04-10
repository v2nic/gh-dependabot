package python

import (
	"os"
	"testing"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

func TestParseUVLock_JSON(t *testing.T) {
	content := `{
		"packages": [
			{"name": "flask", "version": "2.0.0"},
			{"name": "requests", "version": "2.28.0"}
		]
	}`
	tmp, _ := os.CreateTemp("", "uv.lock")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParseUVLock(tmp.Name())
	if err != nil {
		t.Fatalf("ParseUVLock failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}

	expected := "pkg:/python/flask@2.0.0"
	if got := deps["flask@2.0.0"].PackageURL; got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestParseUVLock_TOML(t *testing.T) {
	content := `
[package]
name = "flask"
version = "2.0.0"

[package]
name = "requests"
version = "2.28.0"
`
	tmp, _ := os.CreateTemp("", "uv.lock")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParseUVLock(tmp.Name())
	if err != nil {
		t.Fatalf("ParseUVLock failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}
}

func TestDependencyType(t *testing.T) {
	// Verify return type is compatible with types.Dependency
	var _ map[string]types.Dependency
}
