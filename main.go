package main

import (
	"log"
	"net/http"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/einride/gh-dependabot/cmd"
	"github.com/einride/gh-dependabot/internal/gh"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
)

func main() {
	log.SetFlags(0)
	client := githubv4.NewClient(&http.Client{
		Transport: gh.NewGraphQLRoundTripper(),
	})

	var org string
	var team string
	var securityFilter bool

	runCmd := cobra.Command{
		Use:     "gh dependabot",
		Short:   "Manage Dependabot PRs.",
		Example: "gh dependabot --org einride",
		RunE: func(runCmd *cobra.Command, _ []string) error {
			log.Println("Resolving current user...")
			username, err := gh.Run("api", "graphql", "-f", "query={viewer{login}}", "--jq", ".data.viewer.login")
			if err != nil {
				return err
			}
			query := pullRequestQuery{
				username: username,
				org:      org,
				team:     team,
			}
			log.Printf("Searching \"%s\"...", query.SearchQuery())
			page, err := loadPullRequestPage(client, query)
			if err != nil {
				return err
			}
			pullRequests := page.PullRequests
			for page.HasNextPage {
				log.Printf("Searching \"%s\"... (%d/%d)", query.SearchQuery(), len(pullRequests), page.TotalCount)
				nextPage, err := loadPullRequestPage(client, pullRequestQuery{
					username: username,
					org:      org,
					team:     team,
					cursor:   page.EndCursor,
				})
				if err != nil {
					return err
				}
				pullRequests = append(pullRequests, nextPage.PullRequests...)
				page = nextPage
			}
			if securityFilter {
				log.Printf("Matching pull requests to security alerts...")
				pullRequests, err = filterSecurityPullRequests(runCmd.Context(), client, &pullRequests)
				if err != nil {
					return err
				}
			}
			sort.Slice(pullRequests, func(i, j int) bool {
				return pullRequests[i].updatedAt.Before(pullRequests[j].updatedAt)
			})
			_, err = tea.NewProgram(newApp(client, query, pullRequests), tea.WithAltScreen()).Run()
			return err
		},
	}

	runCmd.AddCommand(cmd.SubmitCmd(), cmd.TriggerCmd())
	runCmd.Flags().StringVarP(&org, "org", "o", "", "organization to query (e.g. einride)")
	runCmd.Flags().StringVarP(&team, "team", "t", "", "team to query (e.g. einride/team-transport-execution)")
	runCmd.Flags().BoolVarP(&securityFilter, "only-security", "s", false, "show only pull requests that relate to security alerts")

	if err := runCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}