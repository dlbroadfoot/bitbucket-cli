package create

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/project/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	Name        string
	Key         string
	Description string
	Workspace   string
	Private     bool
}

func NewCmdCreate(f *cmdutil.Factory, runF func(*CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long: heredoc.Doc(`
			Create a new Bitbucket project.

			Projects are used to organize repositories within a workspace.
		`),
		Example: heredoc.Doc(`
			$ bb project create "My Project" --key MYPROJ --workspace myworkspace
			$ bb project create "My Project" -k MYPROJ -w myworkspace --description "Project description"
			$ bb project create "My Project" -k MYPROJ -w myworkspace --private
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]

			if opts.Key == "" {
				return cmdutil.FlagErrorf("--key is required")
			}

			// If workspace not specified, try to get it from the current repo
			if opts.Workspace == "" {
				repo, err := opts.BaseRepo()
				if err != nil {
					return cmdutil.FlagErrorf("--workspace is required when not in a repository")
				}
				opts.Workspace = repo.RepoWorkspace()
			}

			if runF != nil {
				return runF(opts)
			}
			return createRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Key, "key", "k", "", "Project key (required)")
	cmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Project description")
	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to create the project in")
	cmd.Flags().BoolVar(&opts.Private, "private", false, "Make the project private")

	return cmd
}

func createRun(opts *CreateOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	project, err := createProject(httpClient, opts)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Created project %s (%s)\n",
			cs.SuccessIcon(), cs.Bold(project.Name), cs.Cyan(project.Key))
		fmt.Fprintln(opts.IO.Out, project.HTMLURL())
	}

	return nil
}

type createPayload struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
}

func createProject(client *http.Client, opts *CreateOptions) (*shared.Project, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("workspaces/%s/projects", opts.Workspace)

	payload := createPayload{
		Name:        opts.Name,
		Key:         opts.Key,
		Description: opts.Description,
		IsPrivate:   opts.Private,
	}

	var project shared.Project
	err := apiClient.Post("bitbucket.org", path, payload, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}
