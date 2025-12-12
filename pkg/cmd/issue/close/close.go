package close

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CloseOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
	State       string // closed, resolved, invalid, duplicate, wontfix
}

func NewCmdClose(f *cmdutil.Factory, runF func(*CloseOptions) error) *cobra.Command {
	opts := &CloseOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		State:      "closed",
	}

	cmd := &cobra.Command{
		Use:   "close {<number> | <url>}",
		Short: "Close an issue",
		Long: heredoc.Doc(`
			Close a Bitbucket issue.

			By default, the issue is set to "closed" state. You can specify a different
			closing state using the --state flag.
		`),
		Example: heredoc.Doc(`
			# Close issue #123
			$ bb issue close 123

			# Close issue as resolved
			$ bb issue close 123 --state resolved

			# Close issue as won't fix
			$ bb issue close 123 --state wontfix
		`),
		Args: cmdutil.ExactArgs(1, "cannot close issue: number or url required"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return closeRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "closed", "State to set: closed, resolved, invalid, duplicate, wontfix")

	return cmd
}

func closeRun(opts *CloseOptions) error {
	cs := opts.IO.ColorScheme()

	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the issue argument first to check if it contains repo info
	issueID, issueRepo, err := shared.ParseIssueArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if issueRepo != nil {
		repo = issueRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	// Fetch issue to check current state
	issue, err := list.FetchIssue(httpClient, repo, issueID)
	if err != nil {
		return err
	}

	closedStates := map[string]bool{
		"closed":    true,
		"resolved":  true,
		"invalid":   true,
		"duplicate": true,
		"wontfix":   true,
	}

	if closedStates[issue.State] {
		fmt.Fprintf(opts.IO.ErrOut, "%s Issue #%d (%s) is already closed (%s)\n",
			cs.WarningIcon(), issue.ID, issue.Title, issue.StateDisplay())
		return nil
	}

	// Close the issue
	err = updateIssueState(httpClient, repo, issueID, opts.State)
	if err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s Closed issue #%d (%s) as %s\n",
		cs.SuccessIconWithColor(cs.Red), issue.ID, issue.Title, opts.State)

	return nil
}

// updateIssueState updates an issue's state
// PUT /2.0/repositories/{workspace}/{repo_slug}/issues/{issue_id}
func updateIssueState(client *http.Client, repo bbrepo.Interface, issueID int, state string) error {
	url := fmt.Sprintf("%srepositories/%s/%s/issues/%d",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		issueID,
	)

	payload := map[string]string{
		"state": state,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
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
		return fmt.Errorf("issue #%d not found", issueID)
	}

	if resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	return nil
}
