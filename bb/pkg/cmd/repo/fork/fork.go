package fork

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ForkOptions struct {
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	Repository string
	Workspace  string
	Name       string
	Clone      bool
}

func NewCmdFork(f *cmdutil.Factory, runF func(*ForkOptions) error) *cobra.Command {
	opts := &ForkOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Config:     f.Config,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "fork [<repository>]",
		Short: "Fork a repository",
		Long: heredoc.Doc(`
			Create a fork of a repository.

			Without an argument, creates a fork of the current repository.
			With a WORKSPACE/REPO argument, forks that repository.

			By default, the fork is created in your personal workspace with the same name.
		`),
		Example: heredoc.Doc(`
			# Fork the current repository
			$ bb repo fork

			# Fork a specific repository
			$ bb repo fork owner/repo

			# Fork to a different workspace
			$ bb repo fork owner/repo --workspace my-team

			# Fork with a different name
			$ bb repo fork owner/repo --name my-fork

			# Fork and clone
			$ bb repo fork owner/repo --clone
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Repository = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return forkRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to create the fork in")
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Name for the forked repository")
	cmd.Flags().BoolVarP(&opts.Clone, "clone", "c", false, "Clone the fork after creation")

	return cmd
}

func forkRun(opts *ForkOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()

	var sourceWorkspace, sourceRepo string

	if opts.Repository != "" {
		// Parse provided repository
		parts := strings.SplitN(opts.Repository, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("repository must be in WORKSPACE/REPO format")
		}
		sourceWorkspace = parts[0]
		sourceRepo = parts[1]
	} else {
		// Use current repository
		repo, err := opts.BaseRepo()
		if err != nil {
			return fmt.Errorf("could not determine repository: %w", err)
		}
		sourceWorkspace = repo.RepoWorkspace()
		sourceRepo = repo.RepoSlug()
	}

	// Determine target workspace
	targetWorkspace := opts.Workspace
	if targetWorkspace == "" {
		cfg, err := opts.Config()
		if err != nil {
			return err
		}
		authCfg := cfg.Authentication()
		user, err := authCfg.ActiveUser(bbinstance.Default())
		if err != nil || user == "" {
			return fmt.Errorf("could not determine target workspace: use --workspace flag")
		}
		targetWorkspace = user
	}

	// Determine target name
	targetName := opts.Name
	if targetName == "" {
		targetName = sourceRepo
	}

	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.ErrOut, "Forking %s/%s to %s/%s...\n",
			sourceWorkspace, sourceRepo, targetWorkspace, targetName)
	}

	// Create the fork
	fork, err := forkRepository(httpClient, sourceWorkspace, sourceRepo, targetWorkspace, targetName)
	if err != nil {
		return fmt.Errorf("failed to fork repository: %w", err)
	}

	repoURL := fork.Links.HTML.Href
	cloneURL := ""
	for _, link := range fork.Links.Clone {
		if link.Name == "https" {
			cloneURL = link.Href
			break
		}
	}

	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.Out, "%s Created fork %s/%s\n",
			cs.SuccessIconWithColor(cs.Green),
			targetWorkspace, targetName)
		fmt.Fprintf(opts.IO.Out, "  %s\n", repoURL)
	} else {
		fmt.Fprintln(opts.IO.Out, repoURL)
	}

	if opts.Clone && cloneURL != "" {
		fmt.Fprintf(opts.IO.Out, "\nTo clone this fork:\n  git clone %s\n", cloneURL)
	}

	return nil
}

type forkRepoResponse struct {
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

func forkRepository(client *http.Client, sourceWorkspace, sourceRepo, targetWorkspace, targetName string) (*forkRepoResponse, error) {
	url := fmt.Sprintf("%srepositories/%s/%s/forks",
		api.RESTPrefix(bbinstance.Default()),
		sourceWorkspace,
		sourceRepo,
	)

	payload := map[string]interface{}{
		"workspace": map[string]string{
			"slug": targetWorkspace,
		},
	}

	if targetName != sourceRepo {
		payload["name"] = targetName
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository %s/%s not found", sourceWorkspace, sourceRepo)
	}

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

	var fork forkRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&fork); err != nil {
		return nil, err
	}

	return &fork, nil
}
