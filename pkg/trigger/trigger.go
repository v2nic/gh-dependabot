package trigger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/einride/gh-dependabot/internal/gh"
)

const detectorName = "gh-dependabot-trigger"
const detectorVersion = "1.0.0"
const detectorURL = "https://github.com/v2nic/gh-dependabot"

// Run triggers Dependabot scans on the specified repositories
func Run(repoFlag, orgFlag string, allFlag bool) error {
	repos, err := resolveRepos(repoFlag, orgFlag, allFlag)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories specified")
	}

	for _, r := range repos {
		if err := triggerRepo(r); err != nil {
			return err
		}
	}

	return nil
}

// ResolveRepos determines which repos to target
func ResolveRepos(repoFlag, orgFlag string, allFlag bool) ([]string, error) {
	return resolveRepos(repoFlag, orgFlag, allFlag)
}

// GetRepoState fetches default branch and SHA for a repo
func GetRepoState(owner, name string) (branch, sha string, err error) {
	return getRepoState(owner, name)
}

// =============================================================================
// Internal helpers
// =============================================================================

func resolveRepos(repoFlag, orgFlag string, allFlag bool) ([]string, error) {
	if repoFlag != "" {
		return []string{repoFlag}, nil
	}

	if orgFlag != "" {
		return listOrgRepos(orgFlag)
	}

	if allFlag {
		return listMyRepos()
	}

	return nil, fmt.Errorf("must specify --repo, --org, or --all")
}

func listOrgRepos(org string) ([]string, error) {
	output, err := gh.Run("repo", "list", org, "--json", "nameWithOwner", "--limit", "100", "-q", ".[].nameWithOwner")
	if err != nil {
		return nil, fmt.Errorf("list org repos: %w", err)
	}

	var repos []string
	if err := json.Unmarshal([]byte(output), &repos); err != nil {
		return nil, fmt.Errorf("parse repos: %w", err)
	}

	return repos, nil
}

func listMyRepos() ([]string, error) {
	output, err := gh.Run("repo", "list", "--json", "nameWithOwner", "--limit", "100", "-q", ".[].nameWithOwner")
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	var repos []string
	if err := json.Unmarshal([]byte(output), &repos); err != nil {
		return nil, fmt.Errorf("parse repos: %w", err)
	}

	return repos, nil
}

func triggerRepo(repo string) error {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, name := parts[0], parts[1]

	// Get default branch and actual SHA from GitHub API
	defaultBranch, sha, err := getRepoState(owner, name)
	if err != nil {
		return fmt.Errorf("get repo state: %w", err)
	}

	// Build minimal snapshot with real SHA
	snapshot := map[string]interface{}{
		"version": 0,
		"sha":     sha,
		"ref":     fmt.Sprintf("refs/heads/%s", defaultBranch),
		"job": map[string]string{
			"correlator": fmt.Sprintf("gh-dependabot-trigger-%d", os.Getpid()),
			"id":         fmt.Sprintf("%d", os.Getpid()),
		},
		"detector": map[string]string{
			"name":    detectorName,
			"version": detectorVersion,
			"url":     detectorURL,
		},
		"scanned": "2026-04-08T12:00:00Z",
		"manifests": map[string]interface{}{
			".gitkeep": map[string]interface{}{
				"name": ".gitkeep",
				"file": map[string]string{
					"source_location": ".gitkeep",
				},
				"resolved": map[string]interface{}{},
			},
		},
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	// Use gh api --input for proper JSON handling
	apiCmd := exec.Command("gh", "api", "-X", "POST",
		fmt.Sprintf("repos/%s/%s/dependency-graph/snapshots", owner, name),
		"--input", "-",
	)
	apiCmd.Stdin = bytes.NewReader(payload)

	output, err := apiCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("api call failed: %w\n%s", err, string(output))
	}

	return nil
}

func getRepoState(owner, name string) (branch, sha string, err error) {
	// Get default branch
	output, err := gh.Run("api",
		fmt.Sprintf("repos/%s/%s", owner, name),
		"-q", "{defaultBranch: .default_branch}",
	)
	if err != nil {
		return "main", "", fmt.Errorf("get repo state: %w", err)
	}

	var info struct {
		DefaultBranch string `json:"defaultBranch"`
	}
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return "main", "", err
	}

	// Get the actual commit SHA from the branch
	shaOutput, err := gh.Run("api",
		fmt.Sprintf("repos/%s/%s/git/ref/heads/%s", owner, name, info.DefaultBranch),
		"-q", ".object.sha",
	)
	if err != nil {
		return info.DefaultBranch, "", err
	}

	return info.DefaultBranch, strings.TrimSpace(shaOutput), nil
}
