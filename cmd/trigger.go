package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/einride/gh-dependabot/internal/gh"
	"github.com/spf13/cobra"
)

func TriggerCmd() *cobra.Command {
	var repo string
	var org string
	var all bool

	cmd := &cobra.Command{
		Use:     "trigger",
		Short:   "Trigger Dependabot scans on repositories",
		Aliases: []string{"t"},
		Long: `Trigger Dependabot to scan repositories for security vulnerabilities.

This uses the dependency-graph/snapshots API to submit current dependencies,
which triggers Dependabot to check for known vulnerabilities and create
security update PRs if needed.

Example:
  gh dependabot trigger --repo owner/repo
  gh dependabot trigger --org myorg
  gh dependabot trigger --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repos, err := resolveRepos(repo, org, all)
			if err != nil {
				return err
			}

			if len(repos) == 0 {
				return fmt.Errorf("no repositories specified")
			}

			for _, r := range repos {
				log.Printf("Triggering scan for %s...", r)
				if err := triggerRepo(r); err != nil {
					log.Printf("  Error: %v", err)
					continue
				}
				log.Printf("  Success!")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "r", "", "single repository (owner/repo)")
	cmd.Flags().StringVarP(&org, "org", "o", "", "all repositories in an organization")
	cmd.Flags().BoolVar(&all, "all", false, "trigger for all accessible repositories")

	return cmd
}

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

	// Get default branch
	defaultBranch := "main"
	branchOutput, err := gh.Run("api", fmt.Sprintf("repos/%s/%s", owner, name), "-q", ".default_branch")
	if err == nil {
		defaultBranch = strings.TrimSpace(branchOutput)
	}

	// Create minimal snapshot to trigger scan
	snapshot := map[string]interface{}{
		"version": 0,
		"sha":     "0000000000000000000000000000000000000000", // placeholder
		"ref":     fmt.Sprintf("refs/heads/%s", defaultBranch),
		"job": map[string]string{
			"correlator": fmt.Sprintf("gh-dependabot-trigger-%d", 0),
			"id":         "trigger",
		},
		"detector": map[string]string{
			"name":    "gh-dependabot-trigger",
			"version": "1.0.0",
			"url":     "https://github.com/v2nic/gh-dependabot",
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

	payload, _ := json.Marshal(snapshot)

	// Use direct curl since gh api doesn't handle complex JSON well
	// Use gh api with stdin
	cmd := exec.Command("gh", "api", "-X", "POST",
		fmt.Sprintf("repos/%s/%s/dependency-graph/snapshots", owner, name),
		"--input", "-",
	)
	cmd.Stdin = bytes.NewReader(payload)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("api call failed: %w\n%s", err, string(output))
	}

	return nil
}