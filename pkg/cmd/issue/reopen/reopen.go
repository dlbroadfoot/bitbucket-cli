package reopen

import (
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

type ReopenOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
}

func NewCmdReopen(f *cmdutil.Factory, runF func(*ReopenOptions) error) *cobra.Command {
	opts := &ReopenOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "reopen <number>",
		Short: "Reopen a closed issue",
		Long: heredoc.Doc(`
			Reopen a closed issue.

			This sets the issue state back to "open".
		`),
		Example: heredoc.Doc(`
			$ bb issue reopen 123
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return reopenRun(opts)
		},
	}

	return cmd
}

func reopenRun(opts *ReopenOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the issue argument
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

	// Fetch the current issue to verify it exists and is closed
	issue, err := list.FetchIssue(httpClient, repo, issueID)
	if err != nil {
		return err
	}

	// Check if issue is already open
	if issue.State == "open" || issue.State == "new" {
		return fmt.Errorf("issue #%d is already open", issueID)
	}

	opts.IO.StartProgressIndicator()

	// Update the issue state to open
	err = reopenIssue(httpClient, repo, issueID)

	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Reopened issue #%d\n", cs.SuccessIcon(), issueID)
		fmt.Fprintln(opts.IO.Out, issue.HTMLURL())
	}

	return nil
}

func reopenIssue(client *http.Client, repo bbrepo.Interface, issueID int) error {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/issues/%d",
		repo.RepoWorkspace(), repo.RepoSlug(), issueID)

	payload := map[string]string{"state": "open"}

	return apiClient.Put(repo.RepoHost(), path, payload, nil)
}
