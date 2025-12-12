package view

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	Browser    browser.Browser

	WorkspaceSlug string
	Web           bool
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Browser:    f.Browser,
	}

	cmd := &cobra.Command{
		Use:   "view <workspace>",
		Short: "View a workspace",
		Long: heredoc.Doc(`
			Display details about a Bitbucket workspace.

			Shows workspace information including name, slug, and member count.
		`),
		Example: heredoc.Doc(`
			$ bb workspace view myworkspace
			$ bb workspace view myworkspace --web
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.WorkspaceSlug = args[0]

			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open workspace in the browser")

	return cmd
}

// WorkspaceDetail represents detailed workspace information
type WorkspaceDetail struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Type      string `json:"type"`
	IsPrivate bool   `json:"is_private"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
}

// WorkspaceMembers represents workspace membership stats
type WorkspaceMembers struct {
	Size int `json:"size"`
}

func viewRun(opts *ViewOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	workspace, memberCount, err := fetchWorkspaceDetails(httpClient, opts.WorkspaceSlug)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.Web {
		openURL := workspace.Links.HTML.Href
		if openURL == "" {
			openURL = fmt.Sprintf("https://bitbucket.org/%s", workspace.Slug)
		}
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	return printWorkspaceDetails(opts.IO, workspace, memberCount)
}

func fetchWorkspaceDetails(client *http.Client, slug string) (*WorkspaceDetail, int, error) {
	apiClient := api.NewClientFromHTTP(client)

	// Fetch workspace details
	path := fmt.Sprintf("workspaces/%s", slug)
	var workspace WorkspaceDetail
	err := apiClient.Get("bitbucket.org", path, &workspace)
	if err != nil {
		return nil, 0, err
	}

	// Fetch member count
	memberPath := fmt.Sprintf("workspaces/%s/members?pagelen=1", slug)
	var members WorkspaceMembers
	memberCount := 0
	if err := apiClient.Get("bitbucket.org", memberPath, &members); err == nil {
		memberCount = members.Size
	}

	return &workspace, memberCount, nil
}

func printWorkspaceDetails(io *iostreams.IOStreams, w *WorkspaceDetail, memberCount int) error {
	cs := io.ColorScheme()

	fmt.Fprintf(io.Out, "%s\n", cs.Bold(w.Name))
	fmt.Fprintf(io.Out, "%s\n\n", cs.Gray(w.Slug))

	fmt.Fprintf(io.Out, "Type:     %s\n", w.Type)
	if memberCount > 0 {
		fmt.Fprintf(io.Out, "Members:  %d\n", memberCount)
	}

	if w.Links.HTML.Href != "" {
		fmt.Fprintf(io.Out, "\n%s\n", cs.Gray(w.Links.HTML.Href))
	}

	return nil
}
