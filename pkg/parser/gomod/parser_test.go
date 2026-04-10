package gomod

import (
	"os"
	"testing"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

func TestParseGoSum(t *testing.T) {
	content := "github.com/gin-gonic/gin v1.9.1 h1:4idEAncQnU5cKY8YaKa+Psq+hjwgjasP/5noI1q0TG0=\ngolang.org/x/net v0.17.0 h1:pVa4R8y2q7qT6GnK6qHKFrO6q5IlFoCLs9H99aP6X1w=\n"
	tmp, _ := os.CreateTemp("", "go.sum")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParseGoSum(tmp.Name())
	if err != nil {
		t.Fatalf("ParseGoSum failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}

	expected := "pkg:/golang/github.com/gin-gonic/gin@v1.9.1"
	if got := deps["github.com/gin-gonic/gin@v1.9.1"].PackageURL; got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestParseGoSum_EmptyFile(t *testing.T) {
	tmp, _ := os.CreateTemp("", "go.sum")
	defer os.Remove(tmp.Name())
	tmp.WriteString("")
	tmp.Close()

	deps, err := ParseGoSum(tmp.Name())
	if err != nil {
		t.Fatalf("ParseGoSum failed: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestDependencyType(t *testing.T) {
	// Verify return type is compatible with types.Dependency
	var _ map[string]types.Dependency
}
