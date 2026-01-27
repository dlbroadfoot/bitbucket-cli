package list

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams

	Limit int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces you have access to",
		Long: heredoc.Doc(`
			List all Bitbucket workspaces that you have access to.

			This includes workspaces where you are a member or have been granted
			access to specific repositories.
		`),
		Example: heredoc.Doc(`
			$ bb workspace list
			$ bb workspace list --limit 10
		`),
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of workspaces to list")

	return cmd
}

// Workspace represents a Bitbucket workspace
type Workspace struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Type      string `json:"type"`
	IsPrivate bool   `json:"is_private"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// WorkspaceList represents a paginated list of workspaces
type WorkspaceList struct {
	Size     int         `json:"size"`
	Page     int         `json:"page"`
	PageLen  int         `json:"pagelen"`
	Next     string      `json:"next"`
	Previous string      `json:"previous"`
	Values   []Workspace `json:"values"`
}

func listRun(opts *ListOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	workspaces, err := fetchWorkspaces(httpClient, opts.Limit)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(workspaces) == 0 {
		fmt.Fprintln(opts.IO.Out, "No workspaces found")
		return nil
	}

	return printWorkspaces(opts.IO, workspaces)
}

func fetchWorkspaces(client *http.Client, limit int) ([]Workspace, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("workspaces?pagelen=%d", limit)

	var result WorkspaceList
	err := apiClient.Get("bitbucket.org", path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printWorkspaces(io *iostreams.IOStreams, workspaces []Workspace) error {
	tp := tableprinter.New(io, tableprinter.WithHeader("SLUG", "NAME", "TYPE"))

	for _, w := range workspaces {
		tp.AddField(w.Slug)
		tp.AddField(w.Name)
		tp.AddField(w.Type)
		tp.EndRow()
	}

	return tp.Render()
}
