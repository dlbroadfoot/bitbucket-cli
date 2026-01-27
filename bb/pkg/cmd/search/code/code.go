package code

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CodeOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams

	Query     string
	Workspace string
	Repo      string
	Limit     int
}

func NewCmdCode(f *cmdutil.Factory, runF func(*CodeOptions) error) *cobra.Command {
	opts := &CodeOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "code <query>",
		Short: "Search for code",
		Long: heredoc.Doc(`
			Search for code matching a query.

			The query is matched against file contents in repositories.
			A workspace is required for the search.

			Note: Code search requires the repository to have code search enabled.
		`),
		Example: heredoc.Doc(`
			$ bb search code "func main" --workspace myworkspace
			$ bb search code "import react" --workspace myworkspace --repo myrepo
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
			return codeRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace to search in (required)")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository to search in (optional)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of results")

	return cmd
}

// CodeSearchResult represents a code search result
type CodeSearchResult struct {
	Type         string   `json:"type"`
	ContentMatch string   `json:"content_match"`
	PathMatches  []string `json:"path_matches"`
	File         struct {
		Path  string `json:"path"`
		Type  string `json:"type"`
		Links struct {
			Self struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	} `json:"file"`
}

// CodeSearchResults represents a paginated list of code search results
type CodeSearchResults struct {
	Size              int                `json:"size"`
	Page              int                `json:"page"`
	PageLen           int                `json:"pagelen"`
	Next              string             `json:"next"`
	Previous          string             `json:"previous"`
	Values            []CodeSearchResult `json:"values"`
	SearchQuery       string             `json:"search_query"`
	QuerySubstitution string             `json:"query_substitution"`
}

func codeRun(opts *CodeOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	results, err := searchCode(httpClient, opts.Workspace, opts.Repo, opts.Query, opts.Limit)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintf(opts.IO.Out, "No code found matching %q\n", opts.Query)
		return nil
	}

	return printResults(opts.IO, results)
}

func searchCode(client *http.Client, workspace, repo, query string, limit int) ([]CodeSearchResult, error) {
	apiClient := api.NewClientFromHTTP(client)

	var path string
	encodedQuery := url.QueryEscape(query)

	if repo != "" {
		// Search within a specific repository
		path = fmt.Sprintf("repositories/%s/%s/src?q=%s&pagelen=%d",
			workspace, repo, encodedQuery, limit)
	} else {
		// Search across the workspace using the code search API
		// Note: This is a simplified implementation - Bitbucket's code search
		// API has specific requirements and may not be available for all workspaces
		path = fmt.Sprintf("workspaces/%s/search/code?search_query=%s&pagelen=%d",
			workspace, encodedQuery, limit)
	}

	var result CodeSearchResults
	err := apiClient.Get("bitbucket.org", path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printResults(io *iostreams.IOStreams, results []CodeSearchResult) error {
	cs := io.ColorScheme()

	for _, r := range results {
		// Print file path
		fmt.Fprintf(io.Out, "%s\n", cs.Bold(r.File.Path))

		// Print content match if available
		if r.ContentMatch != "" {
			lines := truncateContent(r.ContentMatch, 3)
			for _, line := range lines {
				fmt.Fprintf(io.Out, "  %s\n", cs.Gray(line))
			}
		}

		fmt.Fprintln(io.Out)
	}

	fmt.Fprintf(io.Out, "%s Showing %d results\n", cs.Gray("─"), len(results))

	return nil
}

// truncateContent limits the content to n lines
func truncateContent(content string, maxLines int) []string {
	var lines []string
	start := 0
	lineCount := 0

	for i, c := range content {
		if c == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
			lineCount++
			if lineCount >= maxLines {
				break
			}
		}
	}

	if start < len(content) && lineCount < maxLines {
		lines = append(lines, content[start:])
	}

	return lines
}
