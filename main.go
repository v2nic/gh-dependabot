package main

import (
	"github.com/einride/gh-dependabot/pkg/submit"
	"github.com/einride/gh-dependabot/pkg/trigger"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gh-dependabot",
		Short: "Manage Dependabot PRs",
		Long: `Manage Dependabot PRs.

Examples:
  gh dependabot --org einride`,
	}

	var onlySecurity bool
	var org, team string

	rootCmd.Flags().BoolVarP(&onlySecurity, "only-security", "s", false, "show only pull requests that relate to security alerts")
	rootCmd.Flags().StringVarP(&org, "org", "o", "", "organization to query (e.g. einride)")
	rootCmd.Flags().StringVarP(&team, "team", "t", "", "team to query (e.g. einride/team-transport-execution)")

	// submit command
	submitCmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit dependencies to trigger Dependabot security updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			lockfile, _ := cmd.Flags().GetString("lockfile")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return submit.Run(repo, lockfile, dryRun)
		},
	}
	submitCmd.Flags().StringP("repo", "r", "", "repository (owner/repo)")
	submitCmd.Flags().StringP("lockfile", "f", "", "path to lockfile (auto-detected if not specified)")
	submitCmd.Flags().Bool("dry-run", false, "preview what would be submitted without making API calls")
	rootCmd.AddCommand(submitCmd)

	// trigger command
	triggerCmd := &cobra.Command{
		Use:     "trigger",
		Aliases: []string{"t"},
		Short:   "Trigger Dependabot scans on repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			orgFlag, _ := cmd.Flags().GetString("org")
			all, _ := cmd.Flags().GetBool("all")
			return trigger.Run(repo, orgFlag, all)
		},
	}
	triggerCmd.Flags().StringP("repo", "r", "", "single repository (owner/repo)")
	triggerCmd.Flags().StringP("org", "o", "", "all repositories in an organization")
	triggerCmd.Flags().Bool("all", false, "trigger for all accessible repositories")
	rootCmd.AddCommand(triggerCmd)

	_ = onlySecurity
	_ = org
	_ = team

	rootCmd.Execute()
}
