package close

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CloseOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
}

func NewCmdClose(f *cmdutil.Factory, runF func(*CloseOptions) error) *cobra.Command {
	opts := &CloseOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "close {<number> | <url>}",
		Short: "Close a pull request",
		Long: heredoc.Doc(`
			Close (decline) a pull request without merging it.

			In Bitbucket, closing a pull request is called "declining".
		`),
		Example: heredoc.Doc(`
			# Close pull request #123
			$ bb pr close 123

			# Close pull request by URL
			$ bb pr close https://bitbucket.org/workspace/repo/pull-requests/123
		`),
		Args: cmdutil.ExactArgs(1, "cannot close pull request: number or url required"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return closeRun(opts)
		},
	}

	return cmd
}

func closeRun(opts *CloseOptions) error {
	cs := opts.IO.ColorScheme()

	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the PR argument first to check if it contains repo info
	prID, prRepo, err := shared.ParsePRArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if prRepo != nil {
		repo = prRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	// Fetch PR to check current state
	pr, err := list.FetchPullRequest(httpClient, repo, prID)
	if err != nil {
		return err
	}

	if pr.State == "MERGED" {
		fmt.Fprintf(opts.IO.ErrOut, "%s Pull request #%d (%s) can't be closed because it was already merged\n",
			cs.FailureIcon(), pr.ID, pr.Title)
		return cmdutil.SilentError
	}

	if pr.State == "DECLINED" {
		fmt.Fprintf(opts.IO.ErrOut, "%s Pull request #%d (%s) is already closed (declined)\n",
			cs.WarningIcon(), pr.ID, pr.Title)
		return nil
	}

	// Decline the PR
	err = declinePR(httpClient, repo, prID)
	if err != nil {
		return fmt.Errorf("failed to close pull request: %w", err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s Closed pull request #%d (%s)\n",
		cs.SuccessIconWithColor(cs.Red), pr.ID, pr.Title)

	return nil
}

// declinePR declines (closes) a pull request
// POST /2.0/repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/decline
func declinePR(client *http.Client, repo bbrepo.Interface, prID int) error {
	url := fmt.Sprintf("%srepositories/%s/%s/pullrequests/%d/decline",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		prID,
	)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("pull request #%d not found", prID)
	}

	if resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	return nil
}
