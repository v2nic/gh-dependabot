package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/einride/gh-dependabot/internal/gh"
	"github.com/einride/gh-dependabot/pkg/parser"
	"github.com/einride/gh-dependabot/pkg/submitter"
	"github.com/spf13/cobra"
)

const detectorName = "gh-dependabot-submit"
const detectorVersion = "1.0.0"
const detectorURL = "https://github.com/v2nic/gh-dependabot"

func SubmitCmd() *cobra.Command {
	var repo string
	var lockfile string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit dependencies to trigger Dependabot security updates",
		Long: `Submit lockfile dependencies to GitHub's dependency graph API.

This triggers Dependabot to check for known vulnerabilities and create
security update PRs if needed.

Example:
  gh dependabot submit --repo owner/repo --lockfile package-lock.json
  gh dependabot submit --lockfile requirements.txt --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine repo
			owner, repoName, err := resolveRepo(repo)
			if err != nil {
				return fmt.Errorf("resolve repo: %w", err)
			}

			// Find lockfile
			lockfilePath, err := findLockfile(lockfile)
			if err != nil {
				return fmt.Errorf("find lockfile: %w", err)
			}

			// Parse lockfile
			parsedDeps, err := parser.ParseLockfile(lockfilePath)
			if err != nil {
				return fmt.Errorf("parse lockfile: %w", err)
			}

			log.Printf("Found %d dependencies", len(parsedDeps))

			if dryRun {
				fmt.Println("Dry run - would submit:")
				for key, dep := range parsedDeps {
					fmt.Printf("  %s -> %s\n", key, dep.PackageURL)
				}
				return nil
			}

			// Convert to submitter format
			deps := make(map[string]submitter.Dependency, len(parsedDeps))
			for k, v := range parsedDeps {
				deps[k] = submitter.Dependency{
					PackageURL:   v.PackageURL,
					Relationship: v.Relationship,
					Scope:        v.Scope,
				}
			}

			// Get default branch and commit SHA from GitHub API
			defaultBranch, sha, err := getRepoState(owner, repoName)
			if err != nil {
				return fmt.Errorf("get repo state: %w", err)
			}

			// Build snapshot
			snapshot := submitter.Snapshot{
				Version: 0,
				SHA:     sha,
				Ref:     "refs/heads/" + defaultBranch,
				Job: submitter.Job{
					Correlator: fmt.Sprintf("gh-dependabot-submit-%d", time.Now().Unix()),
					ID:         fmt.Sprintf("%d", os.Getpid()),
				},
				Detector: submitter.Detector{
					Name:    detectorName,
					Version: detectorVersion,
					URL:     detectorURL,
				},
				Scanned: time.Now().UTC().Format(time.RFC3339),
				Manifests: map[string]submitter.Manifest{
					filepath.Base(lockfilePath): {
						Name: filepath.Base(lockfilePath),
						File: submitter.File{
							SourceLocation: lockfilePath,
						},
						Resolved: deps,
					},
				},
			}

			// Submit to API using --input for proper JSON handling
			log.Printf("Submitting to %s/%s...", owner, repoName)
			payload, err := json.Marshal(snapshot)
			if err != nil {
				return fmt.Errorf("marshal snapshot: %w", err)
			}

			apiCmd := exec.Command("gh", "api", "-X", "POST",
				fmt.Sprintf("repos/%s/%s/dependency-graph/snapshots", owner, repoName),
				"--input", "-",
			)
			apiCmd.Stdin = bytes.NewReader(payload)

			output, err := apiCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("submit: %w\n%s", err, string(output))
			}

			fmt.Println("Successfully submitted dependencies!")
			fmt.Println("Dependabot will now check for vulnerabilities and create security PRs if needed.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "r", "", "repository (owner/repo)")
	cmd.Flags().StringVarP(&lockfile, "lockfile", "f", "", "path to lockfile (auto-detected if not specified)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview what would be submitted without making API calls")

	return cmd
}

func resolveRepo(repoFlag string) (owner, name string, err error) {
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid repo format: %s (expected owner/repo)", repoFlag)
		}
		return parts[0], parts[1], nil
	}

	// Try to get from git remote
	output, err := gh.Run("repo", "view", "--json", "owner, name", "-q", "{owner: .owner.login, name: .name}")
	if err != nil {
		return "", "", fmt.Errorf("could not determine repo: %w", err)
	}

	var repoInfo struct {
		Owner string `json:"owner"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal([]byte(output), &repoInfo); err != nil {
		return "", "", fmt.Errorf("parse repo info: %w", err)
	}

	return repoInfo.Owner, repoInfo.Name, nil
}

func getRepoState(owner, name string) (branch, sha string, err error) {
	// Get default branch and latest commit SHA from API
	output, err := gh.Run("api",
		fmt.Sprintf("repos/%s/%s", owner, name),
		"-q", "{defaultBranch: .default_branch, sha: .default_branch}",
	)
	if err != nil {
		// Fallback to main branch
		return "main", "", fmt.Errorf("get repo state: %w", err)
	}

	var info struct {
		DefaultBranch string `json:"defaultBranch"`
		SHA           string `json:"sha"`
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

func findLockfile(lockfileFlag string) (string, error) {
	if lockfileFlag != "" {
		if _, err := os.Stat(lockfileFlag); err == nil {
			return lockfileFlag, nil
		}
		return "", fmt.Errorf("lockfile not found: %s", lockfileFlag)
	}

	// Auto-detect common lockfiles
	patterns := []string{
		"package-lock.json",
		"pnpm-lock.yaml",
		"yarn.lock",
		"requirements.txt",
		"uv.lock",
		"go.sum",
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for _, pattern := range patterns {
		path := filepath.Join(cwd, pattern)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no lockfile found in %s. Use --lockfile to specify", cwd)
}
