package comment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CommentOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
	Body        string
}

func NewCmdComment(f *cmdutil.Factory, runF func(*CommentOptions) error) *cobra.Command {
	opts := &CommentOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	var bodyFile string

	cmd := &cobra.Command{
		Use:   "comment {<number> | <url>}",
		Short: "Add a comment to an issue",
		Long: heredoc.Doc(`
			Add a comment to a Bitbucket issue.
		`),
		Example: heredoc.Doc(`
			# Add a comment to issue #123
			$ bb issue comment 123 --body "This is a comment"

			# Add a comment from a file
			$ bb issue comment 123 --body-file comment.txt
		`),
		Args: cmdutil.ExactArgs(1, "cannot add comment: issue number or url required"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if bodyFile != "" {
				b, err := cmdutil.ReadFile(bodyFile, opts.IO.In)
				if err != nil {
					return err
				}
				opts.Body = string(b)
			}

			if opts.Body == "" {
				return cmdutil.FlagErrorf("body cannot be empty")
			}

			if runF != nil {
				return runF(opts)
			}
			return commentRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "The comment body text")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read body text from file (use \"-\" to read from standard input)")

	return cmd
}

func commentRun(opts *CommentOptions) error {
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

	err = addIssueComment(httpClient, repo, issueID, opts.Body)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	cs := opts.IO.ColorScheme()
	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.ErrOut, "%s Added comment to issue #%d\n", cs.SuccessIcon(), issueID)
	}

	return nil
}

// addIssueComment adds a comment to an issue
// POST /2.0/repositories/{workspace}/{repo_slug}/issues/{issue_id}/comments
func addIssueComment(client *http.Client, repo bbrepo.Interface, issueID int, body string) error {
	url := fmt.Sprintf("%srepositories/%s/%s/issues/%d/comments",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		issueID,
	)

	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": body,
		},
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	return nil
}
