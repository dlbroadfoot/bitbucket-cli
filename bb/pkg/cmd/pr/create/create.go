package create

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/git"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser
	GitClient  *git.Client

	Title       string
	Body        string
	BaseBranch  string
	HeadBranch  string
	Reviewers   []string
	CloseSource bool
	Web         bool
}

func NewCmdCreate(f *cmdutil.Factory, runF func(*CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
		GitClient:  f.GitClient,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		Long: heredoc.Doc(`
			Create a pull request on Bitbucket.

			When the current branch isn't fully pushed to a git remote, a prompt will ask where
			to push the branch.
		`),
		Example: heredoc.Doc(`
			# Create a pull request interactively
			$ bb pr create

			# Create a pull request with title and body
			$ bb pr create --title "Feature" --body "Description"

			# Create a pull request targeting a specific base branch
			$ bb pr create --base main

			# Create and open in browser
			$ bb pr create --web
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return createRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Title for the pull request")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Body for the pull request")
	cmd.Flags().StringVarP(&opts.BaseBranch, "base", "B", "", "The branch into which you want your code merged")
	cmd.Flags().StringVarP(&opts.HeadBranch, "head", "H", "", "The branch that contains commits for your pull request")
	cmd.Flags().StringSliceVarP(&opts.Reviewers, "reviewer", "r", nil, "Request reviews from people by their username")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", false, "Close source branch after merge")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open the browser to create a pull request")

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

	// Get current branch if head not specified
	headBranch := opts.HeadBranch
	if headBranch == "" {
		headBranch, err = opts.GitClient.CurrentBranch(context.Background())
		if err != nil {
			return fmt.Errorf("could not determine current branch: %w", err)
		}
	}

	// Get default branch if base not specified
	baseBranch := opts.BaseBranch
	if baseBranch == "" {
		baseBranch, err = getDefaultBranch(httpClient, repo)
		if err != nil {
			baseBranch = "main" // fallback
		}
	}

	// Open in browser if requested
	if opts.Web {
		createURL := bbrepo.GenerateRepoURL(repo, "pull-requests/new?source=%s&dest=%s", headBranch, baseBranch)
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(createURL))
		}
		return opts.Browser.Browse(createURL)
	}

	// Validate title
	if opts.Title == "" {
		return fmt.Errorf("title is required. Use --title or --web to create interactively")
	}

	// Create the PR
	pr, err := createPullRequest(httpClient, repo, &createPRParams{
		Title:       opts.Title,
		Description: opts.Body,
		Source:      headBranch,
		Destination: baseBranch,
		Reviewers:   opts.Reviewers,
		CloseSource: opts.CloseSource,
	})
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.Out, "%s Created pull request #%d\n", cs.SuccessIcon(), pr.ID)
	fmt.Fprintf(opts.IO.Out, "%s\n", pr.HTMLURL())

	return nil
}

type createPRParams struct {
	Title       string
	Description string
	Source      string
	Destination string
	Reviewers   []string
	CloseSource bool
}

func createPullRequest(client *http.Client, repo bbrepo.Interface, params *createPRParams) (*shared.PullRequest, error) {
	path := fmt.Sprintf("repositories/%s/%s/pullrequests", repo.RepoWorkspace(), repo.RepoSlug())
	apiURL := api.RESTPrefix(repo.RepoHost()) + path

	// Build reviewers list
	var reviewers []map[string]string
	for _, r := range params.Reviewers {
		reviewers = append(reviewers, map[string]string{"username": r})
	}

	body := map[string]interface{}{
		"title":               params.Title,
		"description":         params.Description,
		"close_source_branch": params.CloseSource,
		"source": map[string]interface{}{
			"branch": map[string]string{
				"name": params.Source,
			},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{
				"name": params.Destination,
			},
		},
	}

	if len(reviewers) > 0 {
		body["reviewers"] = reviewers
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, api.HandleHTTPError(resp)
	}

	var pr shared.PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func getDefaultBranch(client *http.Client, repo bbrepo.Interface) (string, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s", repo.RepoWorkspace(), repo.RepoSlug())

	var result struct {
		MainBranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}

	if err := apiClient.Get(repo.RepoHost(), path, &result); err != nil {
		return "", err
	}

	if result.MainBranch.Name == "" {
		return "main", nil
	}

	return result.MainBranch.Name, nil
}
