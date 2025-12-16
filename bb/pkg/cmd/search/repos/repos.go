package repos

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ReposOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams

	Query     string
	Workspace string
	Limit     int
}

func NewCmdRepos(f *cmdutil.Factory, runF func(*ReposOptions) error) *cobra.Command {
	opts := &ReposOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "repos <query>",
		Short: "Search for repositories",
		Long: heredoc.Doc(`
			Search for repositories matching a query.

			The query is matched against repository names and descriptions.
			A workspace is required for the search.
		`),
		Example: heredoc.Doc(`
			$ bb search repos cli --workspace myworkspace
			$ bb search repos "api client" --workspace myworkspace --limit 10
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Query = args[0]

			if opts.Workspace == "" {
				return cmdutil.FlagErrorf("--workspace is required")
			}

			if runF != nil {
				return runF(opts)
			}
			return reposRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to search in (required)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of results")

	return cmd
}

// Repository represents a Bitbucket repository
type Repository struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	UpdatedOn   string `json:"updated_on"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// RepositoryList represents a paginated list of repositories
type RepositoryList struct {
	Size     int          `json:"size"`
	Page     int          `json:"page"`
	PageLen  int          `json:"pagelen"`
	Next     string       `json:"next"`
	Previous string       `json:"previous"`
	Values   []Repository `json:"values"`
}

func reposRun(opts *ReposOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	repos, err := searchRepositories(httpClient, opts.Workspace, opts.Query, opts.Limit)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(repos) == 0 {
		fmt.Fprintf(opts.IO.Out, "No repositories found matching %q\n", opts.Query)
		return nil
	}

	return printRepositories(opts.IO, repos)
}

func searchRepositories(client *http.Client, workspace, query string, limit int) ([]Repository, error) {
	apiClient := api.NewClientFromHTTP(client)

	// Build the search query
	// The Bitbucket API uses a specific query syntax
	encodedQuery := url.QueryEscape(fmt.Sprintf("name~\"%s\"", query))
	path := fmt.Sprintf("repositories/%s?q=%s&pagelen=%d", workspace, encodedQuery, limit)

	var result RepositoryList
	err := apiClient.Get("bitbucket.org", path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printRepositories(io *iostreams.IOStreams, repos []Repository) error {
	cs := io.ColorScheme()

	tp := tableprinter.New(io, tableprinter.WithHeader("REPOSITORY", "DESCRIPTION", "VISIBILITY", "UPDATED"))

	for _, r := range repos {
		tp.AddField(r.FullName)

		desc := r.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		tp.AddField(desc)

		if r.IsPrivate {
			tp.AddField(cs.Yellow("private"))
		} else {
			tp.AddField(cs.Green("public"))
		}

		if t, err := time.Parse(time.RFC3339, r.UpdatedOn); err == nil {
			tp.AddField(text.FuzzyAgoAbbr(time.Now(), t))
		} else {
			tp.AddField("-")
		}

		tp.EndRow()
	}

	return tp.Render()
}
