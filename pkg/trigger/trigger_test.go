package trigger

import (
	"testing"
)

func TestResolveRepos(t *testing.T) {
	t.Run("single repo", func(t *testing.T) {
		repos, err := ResolveRepos("owner/repo", "", false)
		if err != nil {
			t.Fatalf("ResolveRepos failed: %v", err)
		}
		if len(repos) != 1 || repos[0] != "owner/repo" {
			t.Errorf("expected [owner/repo], got %v", repos)
		}
	})

	t.Run("no flags returns error", func(t *testing.T) {
		_, err := ResolveRepos("", "", false)
		if err == nil {
			t.Error("expected error when no flags provided")
		}
	})

	t.Run("org flag returns repos", func(t *testing.T) {
		// This test is skipped because it requires gh auth
		t.Skip("requires gh authentication")
	})

	t.Run("all flag returns repos", func(t *testing.T) {
		// This test is skipped because it requires gh auth
		t.Skip("requires gh authentication")
	})
}
