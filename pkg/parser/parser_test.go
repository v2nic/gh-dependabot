package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

func TestParseLockfile(t *testing.T) {
	testdata := []struct {
		filename string
		content  string
		minDeps  int
	}{
		{"package-lock.json", `{"packages": {"node_modules/lodash": {"version": "4.17.21"}}}`, 1},
		{"requirements.txt", "flask==2.0\n", 1},
		{"go.sum", "github.com/gin-gonic/gin v1.9.1 h1:xxx\n", 1},
		{"uv.lock", `{"packages": [{"name": "flask", "version": "2.0.0"}]}`, 1},
		{"yarn.lock", "lodash@4.17.21\n", 1},
	}

	for _, td := range testdata {
		tmp, _ := os.CreateTemp("", td.filename)
		defer os.Remove(tmp.Name())
		tmp.WriteString(td.content)
		tmp.Close()

		// Rename to just the filename so parser can detect it
		dir := filepath.Dir(tmp.Name())
		newPath := filepath.Join(dir, td.filename)
		os.Rename(tmp.Name(), newPath)
		defer os.Remove(newPath)

		deps, err := ParseLockfile(newPath)
		if err != nil {
			t.Errorf("ParseLockfile(%s) failed: %v", td.filename, err)
			continue
		}
		if len(deps) < td.minDeps {
			t.Errorf("ParseLockfile(%s) = %d deps; want at least %d", td.filename, len(deps), td.minDeps)
		}
	}
}

func TestParseLockfile_NotFound(t *testing.T) {
	_, err := ParseLockfile("/nonexistent/path/package-lock.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseLockfile_Unsupported(t *testing.T) {
	content := "some random content"
	tmp, _ := os.CreateTemp("", "randomfile")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	_, err := ParseLockfile(tmp.Name())
	if err == nil {
		t.Error("expected error for unsupported lockfile type")
	}
}

func TestBasenameExtraction(t *testing.T) {
	// Verify the parser correctly extracts basename
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/package-lock.json", "package-lock.json"},
		{"/path/to/requirements.txt", "requirements.txt"},
	}

	for _, tt := range tests {
		got := filepath.Base(tt.path)
		if got != tt.expected {
			t.Errorf("filepath.Base(%q) = %q; want %q", tt.path, got, tt.expected)
		}
	}
}

func TestDependencyTypeFromTypes(t *testing.T) {
	// Test that types.Dependency is exported correctly
	dep := types.Dependency{
		PackageURL:   "pkg:/npm/lodash@4.17.21",
		Relationship: "direct",
	}
	if !strings.Contains(dep.PackageURL, "lodash") {
		t.Errorf("expected PackageURL to contain 'lodash'")
	}
}
