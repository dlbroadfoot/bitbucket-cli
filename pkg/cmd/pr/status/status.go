package status

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type StatusOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
}

func NewCmdStatus(f *cmdutil.Factory, runF func(*StatusOptions) error) *cobra.Command {
	opts := &StatusOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of relevant pull requests",
		Long: heredoc.Doc(`
			Show status of pull requests relevant to you.

			This displays:
			- Pull requests created by you
			- Pull requests where you are a reviewer
		`),
		Example: heredoc.Doc(`
			$ bb pr status
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return statusRun(opts)
		},
	}

	return cmd
}

func statusRun(opts *StatusOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	apiClient := api.NewClientFromHTTP(httpClient)

	// Get current user
	currentUser, err := api.CurrentUser(apiClient, repo.RepoHost())
	if err != nil {
		return fmt.Errorf("could not get current user: %w", err)
	}

	opts.IO.StartProgressIndicator()

	// Fetch PRs created by the current user
	createdPRs, err := fetchUserPullRequests(httpClient, repo, currentUser.UUID, "author")
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("could not fetch created PRs: %w", err)
	}

	// Fetch PRs where the current user is a reviewer
	reviewPRs, err := fetchUserPullRequests(httpClient, repo, currentUser.UUID, "reviewer")
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("could not fetch review PRs: %w", err)
	}

	opts.IO.StopProgressIndicator()

	return printStatus(opts.IO, repo, currentUser.DisplayName, createdPRs, reviewPRs)
}

func fetchUserPullRequests(client *http.Client, repo bbrepo.Interface, userUUID string, role string) ([]shared.PullRequest, error) {
	apiClient := api.NewClientFromHTTP(client)

	params := url.Values{}
	params.Set("pagelen", "10")

	// Build query based on role
	var query string
	switch role {
	case "author":
		query = fmt.Sprintf(`author.uuid="%s" AND state="OPEN"`, userUUID)
	case "reviewer":
		query = fmt.Sprintf(`reviewers.uuid="%s" AND state="OPEN"`, userUUID)
	}
	params.Set("q", query)

	path := fmt.Sprintf("repositories/%s/%s/pullrequests?%s",
		repo.RepoWorkspace(), repo.RepoSlug(), params.Encode())

	var result shared.PullRequestList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printStatus(io *iostreams.IOStreams, repo bbrepo.Interface, username string, createdPRs, reviewPRs []shared.PullRequest) error {
	cs := io.ColorScheme()
	out := io.Out

	fmt.Fprintf(out, "\n%s\n\n", cs.Bold(fmt.Sprintf("Pull requests for %s/%s", repo.RepoWorkspace(), repo.RepoSlug())))

	// Created by me
	fmt.Fprintf(out, "%s\n", cs.Bold("Created by you"))
	if len(createdPRs) == 0 {
		fmt.Fprintf(out, "  %s\n", cs.Gray("You have no open pull requests"))
	} else {
		for _, pr := range createdPRs {
			printPRLine(io, &pr)
		}
	}
	fmt.Fprintln(out)

	// Review requests
	fmt.Fprintf(out, "%s\n", cs.Bold("Requesting your review"))
	if len(reviewPRs) == 0 {
		fmt.Fprintf(out, "  %s\n", cs.Gray("No pull requests are requesting your review"))
	} else {
		for _, pr := range reviewPRs {
			printPRLine(io, &pr)
		}
	}
	fmt.Fprintln(out)

	return nil
}

func printPRLine(io *iostreams.IOStreams, pr *shared.PullRequest) {
	cs := io.ColorScheme()
	out := io.Out

	var stateColor func(string) string
	switch pr.State {
	case "OPEN":
		stateColor = cs.Green
	case "MERGED":
		stateColor = cs.Magenta
	case "DECLINED":
		stateColor = cs.Red
	default:
		stateColor = cs.Gray
	}

	fmt.Fprintf(out, "  %s #%d %s %s\n",
		stateColor("•"),
		pr.ID,
		pr.Title,
		cs.Gray(fmt.Sprintf("[%s]", pr.HeadBranch())),
	)
}
