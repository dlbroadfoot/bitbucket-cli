package create

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/internal/bbinstance"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/internal/gh"
	"github.com/cli/bb/v2/pkg/cmd/issue/shared"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	IO         *iostreams.IOStreams
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	BaseRepo   func() (bbrepo.Interface, error)

	Title    string
	Body     string
	Kind     string
	Priority string
	Assignee string
}

func NewCmdCreate(f *cmdutil.Factory, runF func(*CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Config:     f.Config,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long: heredoc.Doc(`
			Create a new issue in a Bitbucket repository.

			The issue tracker must be enabled for the repository.
		`),
		Example: heredoc.Doc(`
			# Create an issue with title
			$ bb issue create --title "Bug in login"

			# Create an issue with title and body
			$ bb issue create --title "Feature request" --body "Add dark mode"

			# Create a bug with critical priority
			$ bb issue create --title "Server crash" --kind bug --priority critical

			# Create and assign an issue
			$ bb issue create --title "Review needed" --assignee username
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Title == "" {
				return cmdutil.FlagErrorf("title is required")
			}

			if runF != nil {
				return runF(opts)
			}
			return createRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Title of the issue (required)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Body of the issue")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "bug", "Kind of issue: {bug|enhancement|proposal|task}")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "major", "Priority: {trivial|minor|major|critical|blocker}")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Assign issue to a user")

	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func createRun(opts *CreateOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	// Build request body
	issueData := map[string]interface{}{
		"title":    opts.Title,
		"kind":     opts.Kind,
		"priority": opts.Priority,
	}

	if opts.Body != "" {
		issueData["content"] = map[string]string{
			"raw": opts.Body,
		}
	}

	if opts.Assignee != "" {
		issueData["assignee"] = map[string]string{
			"username": opts.Assignee,
		}
	}

	jsonBody, err := json.Marshal(issueData)
	if err != nil {
		return err
	}

	// Create issue
	apiURL := fmt.Sprintf("%srepositories/%s/%s/issues",
		bbinstance.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
	)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("issue tracker is not enabled for this repository")
	}

	if resp.StatusCode >= 400 {
		return api.HandleHTTPError(resp)
	}

	var issue shared.Issue
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &issue); err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.Out, "%s Created issue #%d\n", cs.SuccessIcon(), issue.ID)
	fmt.Fprintf(opts.IO.Out, "%s\n", issue.HTMLURL())

	return nil
}
