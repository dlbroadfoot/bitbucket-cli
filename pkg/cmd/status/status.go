package status

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type StatusOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams

	Workspace string
}

func NewCmdStatus(f *cmdutil.Factory, runF func(*StatusOptions) error) *cobra.Command {
	opts := &StatusOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Print information about relevant pull requests and issues",
		Long: heredoc.Doc(`
			Print information about pull requests and issues relevant to you.

			Shows:
			- Pull requests that you created
			- Pull requests awaiting your review
			- Issues assigned to you

			The output is scoped to a specific workspace.
		`),
		Example: heredoc.Doc(`
			$ bb status --workspace myworkspace
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Workspace == "" {
				return cmdutil.FlagErrorf("--workspace is required")
			}

			if runF != nil {
				return runF(opts)
			}
			return statusRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to show status for (required)")

	return cmd
}

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Destination struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"destination"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// Issue represents a Bitbucket issue
type Issue struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	State      string `json:"state"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// PRList represents a paginated list of pull requests
type PRList struct {
	Size   int           `json:"size"`
	Values []PullRequest `json:"values"`
}

// IssueList represents a paginated list of issues
type IssueList struct {
	Size   int     `json:"size"`
	Values []Issue `json:"values"`
}

// User represents a Bitbucket user
type User struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
	AccountID   string `json:"account_id"`
}

func statusRun(opts *StatusOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()

	opts.IO.StartProgressIndicator()

	// Fetch current user first
	apiClient := api.NewClientFromHTTP(httpClient)
	var user User
	if err := apiClient.Get("bitbucket.org", "user", &user); err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Fetch data in parallel
	var wg sync.WaitGroup
	var createdPRs, reviewPRs []PullRequest
	var assignedIssues []Issue
	var createdErr, reviewErr, issuesErr error

	wg.Add(3)

	// Get PRs created by user
	go func() {
		defer wg.Done()
		createdPRs, createdErr = fetchUserPRs(httpClient, opts.Workspace, user.UUID, "author")
	}()

	// Get PRs awaiting user's review
	go func() {
		defer wg.Done()
		reviewPRs, reviewErr = fetchUserPRs(httpClient, opts.Workspace, user.UUID, "reviewer")
	}()

	// Get issues assigned to user
	go func() {
		defer wg.Done()
		assignedIssues, issuesErr = fetchAssignedIssues(httpClient, opts.Workspace, user.UUID)
	}()

	wg.Wait()
	opts.IO.StopProgressIndicator()

	if createdErr != nil {
		return createdErr
	}
	if reviewErr != nil {
		return reviewErr
	}
	if issuesErr != nil {
		// Issues might not be enabled for all repos, ignore errors
		assignedIssues = nil
	}

	// Print results
	hasSomething := false

	// PRs created by user
	if len(createdPRs) > 0 {
		hasSomething = true
		fmt.Fprintf(opts.IO.Out, "%s\n", cs.Bold("Pull Requests Created"))
		for _, pr := range createdPRs {
			stateColor := cs.Gray
			switch pr.State {
			case "OPEN":
				stateColor = cs.Green
			case "MERGED":
				stateColor = cs.Magenta
			case "DECLINED":
				stateColor = cs.Red
			}
			fmt.Fprintf(opts.IO.Out, "  %s #%d %s [%s]\n",
				pr.Destination.Repository.FullName, pr.ID, pr.Title, stateColor(pr.State))
		}
		fmt.Fprintln(opts.IO.Out)
	}

	// PRs awaiting review
	if len(reviewPRs) > 0 {
		hasSomething = true
		fmt.Fprintf(opts.IO.Out, "%s\n", cs.Bold("Pull Requests Awaiting Your Review"))
		for _, pr := range reviewPRs {
			fmt.Fprintf(opts.IO.Out, "  %s #%d %s\n",
				pr.Destination.Repository.FullName, pr.ID, pr.Title)
		}
		fmt.Fprintln(opts.IO.Out)
	}

	// Assigned issues
	if len(assignedIssues) > 0 {
		hasSomething = true
		fmt.Fprintf(opts.IO.Out, "%s\n", cs.Bold("Issues Assigned to You"))
		for _, issue := range assignedIssues {
			stateColor := cs.Gray
			switch issue.State {
			case "new", "open":
				stateColor = cs.Green
			case "resolved":
				stateColor = cs.Magenta
			case "closed":
				stateColor = cs.Gray
			}
			fmt.Fprintf(opts.IO.Out, "  %s #%d %s [%s]\n",
				issue.Repository.FullName, issue.ID, issue.Title, stateColor(issue.State))
		}
		fmt.Fprintln(opts.IO.Out)
	}

	if !hasSomething {
		fmt.Fprintf(opts.IO.Out, "No pull requests or issues found in workspace %s\n", opts.Workspace)
	}

	return nil
}

func fetchUserPRs(client *http.Client, workspace, userUUID, role string) ([]PullRequest, error) {
	apiClient := api.NewClientFromHTTP(client)

	var query string
	if role == "author" {
		query = fmt.Sprintf("state=\"OPEN\" AND author.uuid=\"%s\"", userUUID)
	} else {
		query = fmt.Sprintf("state=\"OPEN\" AND reviewers.uuid=\"%s\"", userUUID)
	}

	path := fmt.Sprintf("pullrequests/%s?q=%s&pagelen=25", workspace, query)

	var result PRList
	err := apiClient.Get("bitbucket.org", path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func fetchAssignedIssues(client *http.Client, workspace, userUUID string) ([]Issue, error) {
	apiClient := api.NewClientFromHTTP(client)

	// This queries for issues assigned to the user across the workspace
	// Note: Bitbucket's issue API requires querying per-repository
	// This is a simplified implementation that queries the workspace level
	query := fmt.Sprintf("assignee.uuid=\"%s\" AND state!=\"closed\" AND state!=\"resolved\"", userUUID)
	path := fmt.Sprintf("repositories/%s?q=has_issues=true&pagelen=10", workspace)

	// Get repos with issues enabled
	type RepoList struct {
		Values []struct {
			FullName string `json:"full_name"`
			Slug     string `json:"slug"`
		} `json:"values"`
	}

	var repos RepoList
	if err := apiClient.Get("bitbucket.org", path, &repos); err != nil {
		return nil, err
	}

	var allIssues []Issue
	for _, repo := range repos.Values {
		issuePath := fmt.Sprintf("repositories/%s/%s/issues?q=%s&pagelen=10",
			workspace, repo.Slug, query)

		var issues IssueList
		if err := apiClient.Get("bitbucket.org", issuePath, &issues); err != nil {
			continue // Skip repos where we can't fetch issues
		}

		for _, issue := range issues.Values {
			issue.Repository.FullName = repo.FullName
			allIssues = append(allIssues, issue)
		}
	}

	return allIssues, nil
}
