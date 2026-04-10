package pip

import (
	"os"
	"testing"

	"github.com/einride/gh-dependabot/pkg/parser/types"
)

func TestParseRequirementsTxt(t *testing.T) {
	content := "requests==2.28.0\nflask>=2.0\ndjango<4.0\n"
	tmp, _ := os.CreateTemp("", "requirements*.txt")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParseRequirementsTxt(tmp.Name())
	if err != nil {
		t.Fatalf("ParseRequirementsTxt failed: %v", err)
	}

	if len(deps) != 3 {
		t.Errorf("expected 3 deps, got %d", len(deps))
	}

	// Check specific deps
	if got := deps["requests@2.28.0"].PackageURL; got != "pkg:/pypi/requests@2.28.0" {
		t.Errorf("requests: expected pkg:/pypi/requests@2.28.0, got %s", got)
	}

	if got := deps["flask@2.0"].PackageURL; got != "pkg:/pypi/flask@2.0" {
		t.Errorf("flask: expected pkg:/pypi/flask@2.0, got %s", got)
	}

	if got := deps["django@4.0"].PackageURL; got != "pkg:/pypi/django@4.0" {
		t.Errorf("django: expected pkg:/pypi/django@4.0, got %s", got)
	}
}

func TestParseRequirementLine(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"flask==2.0", "flask", "2.0"},
		{"requests>=2.28", "requests", "2.28"},
		{"django<4.0", "django", "4.0"},
		{"numpy", "numpy", "latest"},
		{"pandas~=1.5", "pandas", "1.5"},
	}

	for _, tt := range tests {
		name, ver := parseRequirementLine(tt.input)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("parseRequirementLine(%q) = %q, %q; want %q, %q",
				tt.input, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}

func TestParseRequirementsTxt_SkipsFlags(t *testing.T) {
	content := "# This is a comment\n-r other-requirements.txt\n--index-url https://pypi.org/simple\nflask==2.0\n"
	tmp, _ := os.CreateTemp("", "requirements*.txt")
	defer os.Remove(tmp.Name())
	tmp.WriteString(content)
	tmp.Close()

	deps, err := ParseRequirementsTxt(tmp.Name())
	if err != nil {
		t.Fatalf("ParseRequirementsTxt failed: %v", err)
	}

	// Should only have flask, skip comments and flags
	if len(deps) != 1 {
		t.Errorf("expected 1 dep, got %d", len(deps))
	}
}

func TestDependencyType(t *testing.T) {
	// Verify return type is compatible with types.Dependency
	var _ map[string]types.Dependency
}
