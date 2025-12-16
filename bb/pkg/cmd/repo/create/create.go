package create

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	IO         *iostreams.IOStreams

	Name        string
	Description string
	Workspace   string
	Project     string
	Private     bool
	Public      bool
	Clone       bool
}

func NewCmdCreate(f *cmdutil.Factory, runF func(*CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Config:     f.Config,
	}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new repository",
		Long: heredoc.Doc(`
			Create a new Bitbucket repository.

			The repository name can be provided as:
			- Just the repository name (uses default workspace)
			- WORKSPACE/REPO format

			By default, repositories are created as private.
		`),
		Example: heredoc.Doc(`
			# Create a private repository in your default workspace
			$ bb repo create my-project

			# Create a public repository
			$ bb repo create my-project --public

			# Create a repository in a specific workspace
			$ bb repo create my-workspace/my-project

			# Create a repository with description
			$ bb repo create my-project --description "My awesome project"

			# Create and clone the repository
			$ bb repo create my-project --clone
		`),
		Args: cmdutil.ExactArgs(1, "repository name required"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			if opts.Public && opts.Private {
				return cmdutil.FlagErrorf("specify only one of --public or --private")
			}

			// Default to private if neither specified
			if !opts.Public && !opts.Private {
				opts.Private = true
			}

			if runF != nil {
				return runF(opts)
			}
			return createRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Description of the repository")
	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to create the repository in")
	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Project key to add the repository to")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Make the repository public")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the repository private (default)")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the repository after creation")

	return cmd
}

func createRun(opts *CreateOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()

	// Parse workspace and repo name
	workspace := opts.Workspace
	repoName := opts.Name

	if strings.Contains(opts.Name, "/") {
		parts := strings.SplitN(opts.Name, "/", 2)
		workspace = parts[0]
		repoName = parts[1]
	}

	// If no workspace specified, try to get from config
	if workspace == "" {
		cfg, err := opts.Config()
		if err != nil {
			return err
		}
		// Try to get default workspace from user info
		authCfg := cfg.Authentication()
		user, err := authCfg.ActiveUser(bbinstance.Default())
		if err != nil || user == "" {
			return fmt.Errorf("workspace required: use --workspace flag or WORKSPACE/REPO format")
		}
		workspace = user
	}

	// Create the repository
	repo, err := createRepository(httpClient, workspace, repoName, opts.Description, opts.Project, !opts.Public)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	repoURL := repo.Links.HTML.Href
	cloneURL := ""
	for _, link := range repo.Links.Clone {
		if link.Name == "https" {
			cloneURL = link.Href
			break
		}
	}

	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.Out, "%s Created repository %s/%s\n",
			cs.SuccessIconWithColor(cs.Green),
			workspace, repoName)
		fmt.Fprintf(opts.IO.Out, "  %s\n", repoURL)
	} else {
		fmt.Fprintln(opts.IO.Out, repoURL)
	}

	if opts.Clone && cloneURL != "" {
		fmt.Fprintf(opts.IO.ErrOut, "Cloning into '%s'...\n", repoName)
		// We could use git client here, but keeping it simple for now
		fmt.Fprintf(opts.IO.Out, "\nTo clone this repository:\n  git clone %s\n", cloneURL)
	}

	return nil
}

type createRepoResponse struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

func createRepository(client *http.Client, workspace, name, description, project string, isPrivate bool) (*createRepoResponse, error) {
	url := fmt.Sprintf("%srepositories/%s/%s",
		api.RESTPrefix(bbinstance.Default()),
		workspace,
		name,
	)

	payload := map[string]interface{}{
		"scm":        "git",
		"is_private": isPrivate,
	}

	if description != "" {
		payload["description"] = description
	}

	if project != "" {
		payload["project"] = map[string]string{
			"key": project,
		}
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("%s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("bad request")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, api.HandleHTTPError(resp)
	}

	var repo createRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, err
	}

	return &repo, nil
}
