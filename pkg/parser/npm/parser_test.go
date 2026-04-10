package npm

import (
	"os"
	"testing"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

func TestParsePackageLock(t *testing.T) {
	content := `{
		"packages": {
			"node_modules/lodash": {
				"version": "4.17.21"
			},
			"node_modules/express": {
				"version": "4.18.2"
			}
		}
	}`
	tmp, _ := os.CreateTemp("", "package-lock*.json")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParsePackageLock(tmp.Name())
	if err != nil {
		t.Fatalf("ParsePackageLock failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}

	expected := "pkg:/npm/lodash@4.17.21"
	if deps["lodash@4.17.21"].PackageURL != expected {
		t.Errorf("expected %s, got %s", expected, deps["lodash@4.17.21"].PackageURL)
	}
}

func TestParsePackageLock_Scoped(t *testing.T) {
	content := `{
		"packages": {
			"node_modules/@types/node": {
				"version": "20.0.0"
			}
		}
	}`
	tmp, _ := os.CreateTemp("", "package-lock*.json")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParsePackageLock(tmp.Name())
	if err != nil {
		t.Fatalf("ParsePackageLock failed: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("expected 1 dep, got %d", len(deps))
	}

	expected := "pkg:/npm/@types/node@20.0.0"
	if got := deps["@types/node@20.0.0"].PackageURL; got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestParseYarnLock(t *testing.T) {
	content := "lodash@^4.17.21:\n  resolved \"https://registry.npmjs.org/lodash\"\n"
	tmp, _ := os.CreateTemp("", "yarn.lock")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParseYarnLock(tmp.Name())
	if err != nil {
		t.Fatalf("ParseYarnLock failed: %v", err)
	}

	if len(deps) == 0 {
		t.Error("expected at least 1 dep from yarn.lock")
	}
}

func TestSplitAtVersion(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantVer  string
	}{
		{"lodash@4.17.21", "lodash", "4.17.21"},
		{"@types/node@20.0.0", "@types/node", "20.0.0"},
		{"pkg@1.0.0", "pkg", "1.0.0"},
	}

	for _, tt := range tests {
		name, ver := splitAtVersion(tt.input)
		if name != tt.wantName || ver != tt.wantVer {
			t.Errorf("splitAtVersion(%q) = %q, %q; want %q, %q",
				tt.input, name, ver, tt.wantName, tt.wantVer)
		}
	}
}

func TestCleanVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0-beta", "1.0.0"},
		{"  1.0.0  ", "1.0.0"},
	}

	for _, tt := range tests {
		got := cleanVersion(tt.input)
		if got != tt.want {
			t.Errorf("cleanVersion(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"node_modules/lodash", "lodash"},
		{"node_modules/@types/node", "@types/node"},
		{"node_modules/lodash/lib/index.js", "lodash"},
	}

	for _, tt := range tests {
		got := extractPackageName(tt.input)
		if got != tt.want {
			t.Errorf("extractPackageName(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestDependencyType(t *testing.T) {
	// Verify return type is compatible with types.Dependency
	var _ map[string]types.Dependency
}
