package list

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/project/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	Workspace string
	Limit     int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List projects in a workspace",
		Aliases: []string{"ls"},
		Long: heredoc.Doc(`
			List projects in a Bitbucket workspace.

			Projects are used to organize repositories within a workspace.
		`),
		Example: heredoc.Doc(`
			$ bb project list --workspace myworkspace
			$ bb project list -w myworkspace --limit 50
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return listRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to list projects from")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of projects to list")

	return cmd
}

func listRun(opts *ListOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	projects, err := fetchProjects(httpClient, opts.Workspace, opts.Limit)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(projects) == 0 {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "No projects found in workspace %s\n", opts.Workspace)
		}
		return nil
	}

	return printProjects(opts.IO, projects)
}

func fetchProjects(client *http.Client, workspace string, limit int) ([]shared.Project, error) {
	apiClient := api.NewClientFromHTTP(client)

	params := url.Values{}
	if limit > 0 {
		params.Set("pagelen", fmt.Sprintf("%d", limit))
	}

	path := fmt.Sprintf("workspaces/%s/projects?%s", workspace, params.Encode())

	var result shared.ProjectList
	err := apiClient.Get("bitbucket.org", path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printProjects(io *iostreams.IOStreams, projects []shared.Project) error {
	cs := io.ColorScheme()

	tp := tableprinter.New(io, tableprinter.WithHeader("KEY", "NAME", "DESCRIPTION"))

	for _, p := range projects {
		tp.AddField(cs.Bold(p.Key))
		tp.AddField(p.Name)

		desc := p.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		tp.AddField(cs.Gray(desc))

		tp.EndRow()
	}

	return tp.Render()
}
