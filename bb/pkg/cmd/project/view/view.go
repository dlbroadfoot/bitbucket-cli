package view

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/project/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser

	ProjectKey string
	Workspace  string
	Web        bool
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
	}

	cmd := &cobra.Command{
		Use:   "view <project-key>",
		Short: "View a project",
		Long: heredoc.Doc(`
			View details of a Bitbucket project.

			With --web, open the project in a web browser instead.
		`),
		Example: heredoc.Doc(`
			$ bb project view PROJ --workspace myworkspace
			$ bb project view PROJ -w myworkspace --web
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ProjectKey = args[0]

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
			return viewRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace the project belongs to")
	cmd.Flags().BoolVarP(&opts.Web, "web", "", false, "Open project in the browser")

	return cmd
}

func viewRun(opts *ViewOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	project, err := fetchProject(httpClient, opts.Workspace, opts.ProjectKey)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	openURL := project.HTMLURL()

	if opts.Web {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	return printProject(opts.IO, project)
}

func fetchProject(client *http.Client, workspace, projectKey string) (*shared.Project, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("workspaces/%s/projects/%s", workspace, projectKey)

	var project shared.Project
	err := apiClient.Get("bitbucket.org", path, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func printProject(io *iostreams.IOStreams, project *shared.Project) error {
	cs := io.ColorScheme()
	out := io.Out

	// Name and key
	fmt.Fprintf(out, "%s (%s)\n", cs.Bold(project.Name), cs.Cyan(project.Key))
	fmt.Fprintln(out)

	// Description
	if project.Description != "" {
		fmt.Fprintln(out, project.Description)
		fmt.Fprintln(out)
	}

	// Visibility
	visibility := "Public"
	if project.IsPrivate {
		visibility = "Private"
	}
	fmt.Fprintf(out, "%s %s\n", cs.Bold("Visibility:"), visibility)

	// URL
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", cs.Gray(project.HTMLURL()))

	return nil
}
