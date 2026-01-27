package edit

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type EditOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	RepoArg     string
	Name        string
	Description string
	Website     string
	Language    string
	Private     *bool
	ForkPolicy  string
	MainBranch  string
	Project     string
}

func NewCmdEdit(f *cmdutil.Factory, runF func(*EditOptions) error) *cobra.Command {
	opts := &EditOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	var setPrivate, setPublic bool

	cmd := &cobra.Command{
		Use:   "edit [<repository>]",
		Short: "Edit repository settings",
		Long: heredoc.Doc(`
			Edit settings for a repository.

			Without an argument, edits the current repository.
		`),
		Example: heredoc.Doc(`
			$ bb repo edit --description "New description"
			$ bb repo edit --private
			$ bb repo edit --public
			$ bb repo edit myworkspace/myrepo --main-branch develop
			$ bb repo edit --project PROJ
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RepoArg = args[0]
			}

			flags := cmd.Flags()

			// Handle privacy flags
			if flags.Changed("private") {
				opts.Private = &setPrivate
			} else if flags.Changed("public") {
				notPublic := !setPublic
				opts.Private = &notPublic
			}

			// Validate that at least one edit flag is provided
			hasEdits := opts.Name != "" ||
				opts.Description != "" ||
				opts.Website != "" ||
				opts.Language != "" ||
				opts.Private != nil ||
				opts.ForkPolicy != "" ||
				opts.MainBranch != "" ||
				opts.Project != ""

			if !hasEdits {
				return cmdutil.FlagErrorf("at least one edit flag is required")
			}

			// Validate fork policy
			if opts.ForkPolicy != "" {
				validPolicies := []string{"allow_forks", "no_public_forks", "no_forks"}
				valid := false
				for _, p := range validPolicies {
					if opts.ForkPolicy == p {
						valid = true
						break
					}
				}
				if !valid {
					return cmdutil.FlagErrorf("invalid fork policy %q, valid options: allow_forks, no_public_forks, no_forks", opts.ForkPolicy)
				}
			}

			if runF != nil {
				return runF(opts)
			}
			return editRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Rename the repository")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Set the repository description")
	cmd.Flags().StringVar(&opts.Website, "website", "", "Set the repository website URL")
	cmd.Flags().StringVarP(&opts.Language, "language", "l", "", "Set the repository language")
	cmd.Flags().BoolVar(&setPrivate, "private", false, "Make the repository private")
	cmd.Flags().BoolVar(&setPublic, "public", false, "Make the repository public")
	cmd.Flags().StringVar(&opts.ForkPolicy, "fork-policy", "", "Set fork policy (allow_forks, no_public_forks, no_forks)")
	cmd.Flags().StringVar(&opts.MainBranch, "main-branch", "", "Set the default branch")
	cmd.Flags().StringVarP(&opts.Project, "project", "p", "", "Move repository to a project by key")

	return cmd
}

func editRun(opts *EditOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	var repo bbrepo.Interface
	if opts.RepoArg != "" {
		repo, err = bbrepo.FromFullName(opts.RepoArg)
		if err != nil {
			return err
		}
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	opts.IO.StartProgressIndicator()
	err = updateRepository(httpClient, repo, opts)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Edited repository %s/%s\n",
			cs.SuccessIcon(), repo.RepoWorkspace(), repo.RepoSlug())
	}

	return nil
}

type repoUpdatePayload struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Website     string      `json:"website,omitempty"`
	Language    string      `json:"language,omitempty"`
	IsPrivate   *bool       `json:"is_private,omitempty"`
	ForkPolicy  string      `json:"fork_policy,omitempty"`
	MainBranch  *mainBranch `json:"mainbranch,omitempty"`
	Project     *projectRef `json:"project,omitempty"`
}

type mainBranch struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type projectRef struct {
	Key string `json:"key"`
}

func updateRepository(client *http.Client, repo bbrepo.Interface, opts *EditOptions) error {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s", repo.RepoWorkspace(), repo.RepoSlug())

	payload := repoUpdatePayload{}

	if opts.Name != "" {
		payload.Name = opts.Name
	}
	if opts.Description != "" {
		payload.Description = opts.Description
	}
	if opts.Website != "" {
		payload.Website = opts.Website
	}
	if opts.Language != "" {
		payload.Language = opts.Language
	}
	if opts.Private != nil {
		payload.IsPrivate = opts.Private
	}
	if opts.ForkPolicy != "" {
		payload.ForkPolicy = opts.ForkPolicy
	}
	if opts.MainBranch != "" {
		payload.MainBranch = &mainBranch{
			Name: opts.MainBranch,
			Type: "branch",
		}
	}
	if opts.Project != "" {
		payload.Project = &projectRef{Key: opts.Project}
	}

	return apiClient.Put(repo.RepoHost(), path, payload, nil)
}
