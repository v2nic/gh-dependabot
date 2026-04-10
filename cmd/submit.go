package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
			deps, err := parser.ParseLockfile(lockfilePath)
			if err != nil {
				return fmt.Errorf("parse lockfile: %w", err)
			}

			// Convert to submitter format
			parsedDeps := make(map[string]submitter.Dependency)
			for key, dep := range deps {
				parsedDeps[key] = submitter.Dependency{
					PackageURL:   dep.PackageURL,
					Relationship: dep.Relationship,
					Scope:        dep.Scope,
				}
			}

			log.Printf("Found %d dependencies", len(parsedDeps))

			if dryRun {
				fmt.Println("Dry run - would submit:")
				for key, dep := range parsedDeps {
					fmt.Printf("  %s -> %s\n", key, dep.PackageURL)
				}
				return nil
			}

			// Get current commit SHA
			sha, err := gh.Run("rev-parse", "HEAD")
			if err != nil {
				// Fallback to get from remote
				sha = "0000000000000000000000000000000000000000"
			}

			// Get default branch
			branch, err := gh.Run("branch", "--show-current")
			if err != nil {
				branch = "main"
			}
			if branch == "" {
				branch = "main"
			}

			// Build snapshot
			snapshot := submitter.Snapshot{
				Version: 0,
				SHA:     sha,
				Ref:     "refs/heads/" + branch,
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
						Resolved: parsedDeps,
					},
				},
			}

			// Submit to API
			log.Printf("Submitting to %s/%s...", owner, repoName)
			payload, _ := json.Marshal(snapshot)
			output, err := gh.Run("api", "-X", "POST",
				fmt.Sprintf("repos/%s/%s/dependency-graph/snapshots", owner, repoName),
				"-H", "Accept: application/vnd.github+json",
				"-f", fmt.Sprintf("snapshot=%s", string(payload)),
			)
			if err != nil {
				return fmt.Errorf("submit: %w\n%s", err, output)
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
		// Parse owner/repo
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
		"package.json", // will use deps from package.json
		"pnpm-lock.yaml",
		"yarn.lock",
		"requirements.txt",
		"Pipfile.lock",
		"pyproject.lock",
		"uv.lock",
		"go.sum",
		"Cargo.lock",
		"Gemfile.lock",
		"composer.lock",
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