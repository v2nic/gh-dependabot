package submit

import (
	"testing"
)

func TestResolveRepo(t *testing.T) {
	tests := []struct {
		input    string
		wantOwn  string
		wantName string
		wantErr  bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"my-org/my-repo", "my-org", "my-repo", false},
	}

	for _, tt := range tests {
		owner, name, err := ResolveRepo(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("ResolveRepo(%q) expected error, got none", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ResolveRepo(%q) unexpected error: %v", tt.input, err)
		}
		if owner != tt.wantOwn || name != tt.wantName {
			t.Errorf("ResolveRepo(%q) = (%q, %q), want (%q, %q)",
				tt.input, owner, name, tt.wantOwn, tt.wantName)
		}
	}
}

func TestResolveRepo_SinglePart(t *testing.T) {
	// Single part without slash should fail
	_, _, err := ResolveRepo("owner")
	if err == nil {
		t.Error("expected error for single-part repo")
	}
}

func TestFindLockfile(t *testing.T) {
	// Test with non-existent file - should return error
	_, err := FindLockfile("/nonexistent/path/package-lock.json")
	if err == nil {
		t.Error("expected error for nonexistent lockfile")
	}
}
