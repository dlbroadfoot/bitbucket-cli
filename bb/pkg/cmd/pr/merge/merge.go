package merge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/pkg/cmd/pr/list"
	"github.com/cli/bb/v2/pkg/cmd/pr/shared"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type MergeOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg   string
	MergeStrategy string
	Message       string
	CloseSource   bool
}

func NewCmdMerge(f *cmdutil.Factory, runF func(*MergeOptions) error) *cobra.Command {
	opts := &MergeOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "merge [<number> | <url>]",
		Short: "Merge a pull request",
		Long: heredoc.Doc(`
			Merge a pull request on Bitbucket.

			By default, the merge strategy is "merge commit". Use --squash or --ff to change.
		`),
		Example: heredoc.Doc(`
			# Merge pull request #123
			$ bb pr merge 123

			# Squash merge
			$ bb pr merge 123 --squash

			# Merge with a custom message
			$ bb pr merge 123 --message "Merge feature branch"

			# Merge and close source branch
			$ bb pr merge 123 --close-source
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return mergeRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.MergeStrategy, "strategy", "s", "merge_commit", "Merge strategy: merge_commit, squash, fast_forward")
	cmd.Flags().StringVarP(&opts.Message, "message", "m", "", "Commit message for the merge")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", false, "Close source branch after merge")

	// Convenience flags
	cmd.Flags().Bool("squash", false, "Use squash merge strategy")
	cmd.Flags().Bool("ff", false, "Use fast-forward merge strategy")

	return cmd
}

func mergeRun(opts *MergeOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	// Parse the PR argument
	prID, prRepo, err := shared.ParsePRArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided
	if prRepo != nil {
		repo = prRepo
	}

	// Fetch the PR first to check state
	pr, err := list.FetchPullRequest(httpClient, repo, prID)
	if err != nil {
		return err
	}

	if pr.State != "OPEN" {
		return fmt.Errorf("pull request #%d is not open (state: %s)", prID, pr.StateDisplay())
	}

	// Perform the merge
	err = mergePullRequest(httpClient, repo, prID, opts)
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.Out, "%s Merged pull request #%d\n", cs.SuccessIcon(), prID)

	return nil
}

func mergePullRequest(client *http.Client, repo bbrepo.Interface, prID int, opts *MergeOptions) error {
	path := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/merge",
		repo.RepoWorkspace(), repo.RepoSlug(), prID)
	apiURL := api.RESTPrefix(repo.RepoHost()) + path

	body := map[string]interface{}{
		"close_source_branch": opts.CloseSource,
	}

	if opts.MergeStrategy != "" {
		body["merge_strategy"] = opts.MergeStrategy
	}

	if opts.Message != "" {
		body["message"] = opts.Message
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	return nil
}
