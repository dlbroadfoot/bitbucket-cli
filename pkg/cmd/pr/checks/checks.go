package checks

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ChecksOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser

	SelectorArg string
	Web         bool
}

func NewCmdChecks(f *cmdutil.Factory, runF func(*ChecksOptions) error) *cobra.Command {
	opts := &ChecksOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
	}

	cmd := &cobra.Command{
		Use:   "checks [<number> | <url>]",
		Short: "Show CI status for a pull request",
		Long: heredoc.Doc(`
			Show CI status (pipeline builds) for a pull request.

			Without an argument, shows checks for the PR associated with the current branch.
		`),
		Example: heredoc.Doc(`
			$ bb pr checks 123
			$ bb pr checks 123 --web
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return checksRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open PR checks in the browser")

	return cmd
}

// BuildStatus represents a commit build status
type BuildStatus struct {
	Key         string `json:"key"`
	State       string `json:"state"` // SUCCESSFUL, FAILED, INPROGRESS, STOPPED
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
}

// BuildStatusList represents a paginated list of build statuses
type BuildStatusList struct {
	Size     int           `json:"size"`
	Page     int           `json:"page"`
	PageLen  int           `json:"pagelen"`
	Next     string        `json:"next"`
	Previous string        `json:"previous"`
	Values   []BuildStatus `json:"values"`
}

func checksRun(opts *ChecksOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the PR argument
	prID, prRepo, err := shared.ParsePRArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if prRepo != nil {
		repo = prRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	// Fetch the PR to get the source commit
	opts.IO.StartProgressIndicator()
	pr, err := list.FetchPullRequest(httpClient, repo, prID)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return err
	}

	commitHash := pr.Source.Commit.Hash

	// Fetch build statuses for the commit
	statuses, err := fetchBuildStatuses(httpClient, repo, commitHash)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.Web {
		// Open the PR page which shows the checks
		openURL := pr.HTMLURL()
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	if len(statuses) == 0 {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.Out, "No CI checks found for PR #%d\n", prID)
		}
		return nil
	}

	return printStatuses(opts.IO, statuses)
}

func fetchBuildStatuses(client *http.Client, repo bbrepo.Interface, commitHash string) ([]BuildStatus, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/commit/%s/statuses?pagelen=100",
		repo.RepoWorkspace(), repo.RepoSlug(), commitHash)

	var result BuildStatusList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printStatuses(io *iostreams.IOStreams, statuses []BuildStatus) error {
	cs := io.ColorScheme()

	tp := tableprinter.New(io, tableprinter.WithHeader("STATUS", "NAME", "DESCRIPTION", "UPDATED"))

	// Count results
	var passed, failed, pending int

	for _, s := range statuses {
		// Status with color
		var status string
		var statusColor func(string) string
		switch s.State {
		case "SUCCESSFUL":
			status = "pass"
			statusColor = cs.Green
			passed++
		case "FAILED":
			status = "fail"
			statusColor = cs.Red
			failed++
		case "INPROGRESS":
			status = "pending"
			statusColor = cs.Yellow
			pending++
		case "STOPPED":
			status = "stopped"
			statusColor = cs.Gray
		default:
			status = s.State
			statusColor = cs.Gray
		}
		tp.AddField(statusColor(status))

		// Name
		name := s.Name
		if name == "" {
			name = s.Key
		}
		tp.AddField(name)

		// Description
		desc := s.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		tp.AddField(desc)

		// Updated
		if t, err := time.Parse(time.RFC3339, s.UpdatedOn); err == nil {
			tp.AddField(text.FuzzyAgo(time.Now(), t))
		} else {
			tp.AddField("-")
		}

		tp.EndRow()
	}

	if err := tp.Render(); err != nil {
		return err
	}

	// Summary
	fmt.Fprintln(io.Out)
	if failed > 0 {
		fmt.Fprintf(io.Out, "%s %d/%d checks passing\n",
			cs.Red("✗"), passed, len(statuses))
	} else if pending > 0 {
		fmt.Fprintf(io.Out, "%s %d/%d checks passing, %d pending\n",
			cs.Yellow("●"), passed, len(statuses), pending)
	} else {
		fmt.Fprintf(io.Out, "%s All checks passed\n", cs.Green("✓"))
	}

	return nil
}
