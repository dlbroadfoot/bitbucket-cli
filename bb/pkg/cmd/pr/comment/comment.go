package comment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/pkg/cmd/pr/shared"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CommentOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
	Body        string
	BodyFile    string
}

func NewCmdComment(f *cmdutil.Factory, runF func(*CommentOptions) error) *cobra.Command {
	opts := &CommentOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "comment [<number> | <url>]",
		Short: "Add a comment to a pull request",
		Long: heredoc.Doc(`
			Add a comment to a Bitbucket pull request.

			Without the body text supplied through flags, the command will interactively
			prompt for the comment text.
		`),
		Example: heredoc.Doc(`
			# Add a comment to pull request #123
			$ bb pr comment 123 --body "Looks good to me!"

			# Add a comment using a file
			$ bb pr comment 123 --body-file comment.txt

			# Add a comment to a PR by URL
			$ bb pr comment https://bitbucket.org/workspace/repo/pull-requests/123 --body "LGTM"
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if opts.Body == "" && opts.BodyFile == "" {
				return cmdutil.FlagErrorf("body text is required; use --body or --body-file")
			}

			if opts.Body != "" && opts.BodyFile != "" {
				return cmdutil.FlagErrorf("specify only one of --body or --body-file")
			}

			if opts.BodyFile != "" {
				b, err := cmdutil.ReadFile(opts.BodyFile, opts.IO.In)
				if err != nil {
					return err
				}
				opts.Body = string(b)
			}

			if runF != nil {
				return runF(opts)
			}
			return commentRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "The comment body text")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from file (use \"-\" to read from standard input)")

	return cmd
}

func commentRun(opts *CommentOptions) error {
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

	// Create the comment
	err = createComment(httpClient, repo, prID, opts.Body)
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.ErrOut, "%s Added comment to pull request #%d\n", cs.SuccessIcon(), prID)

	return nil
}

func createComment(client *http.Client, repo bbrepo.Interface, prID int, body string) error {
	path := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments",
		repo.RepoWorkspace(), repo.RepoSlug(), prID)

	apiURL := api.RESTPrefix(repo.RepoHost()) + path

	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": body,
		},
	}

	jsonBody, err := json.Marshal(payload)
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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	return nil
}
