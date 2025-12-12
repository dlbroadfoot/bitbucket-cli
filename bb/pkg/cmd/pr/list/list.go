package list

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/internal/tableprinter"
	"github.com/cli/bb/v2/pkg/cmd/pr/shared"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	State  string
	Author string
	Limit  int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests in a repository",
		Long: heredoc.Doc(`
			List pull requests in a Bitbucket repository.

			By default, only open pull requests are listed. Use --state to filter by state.
		`),
		Example: heredoc.Doc(`
			# List open pull requests
			$ bb pr list

			# List all pull requests
			$ bb pr list --state all

			# List merged pull requests
			$ bb pr list --state merged

			# List pull requests by a specific author
			$ bb pr list --author username
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

	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: {open|merged|declined|all}")
	cmd.Flags().StringVarP(&opts.Author, "author", "a", "", "Filter by author username")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of pull requests to list")

	return cmd
}

func listRun(opts *ListOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	prs, err := fetchPullRequests(httpClient, repo, opts)
	if err != nil {
		return err
	}

	if len(prs) == 0 {
		return cmdutil.NewNoResultsError(fmt.Sprintf("no pull requests match your search in %s", bbrepo.FullName(repo)))
	}

	return printPullRequests(opts.IO, prs)
}

func printPullRequests(io *iostreams.IOStreams, prs []shared.PullRequest) error {
	cs := io.ColorScheme()
	tp := tableprinter.New(io, tableprinter.WithHeader("ID", "TITLE", "BRANCH", "AUTHOR", "STATE"))

	for _, pr := range prs {
		prNum := strconv.Itoa(pr.ID)

		var stateColor func(string) string
		switch pr.State {
		case "OPEN":
			stateColor = cs.Green
		case "MERGED":
			stateColor = cs.Magenta
		case "DECLINED":
			stateColor = cs.Red
		default:
			stateColor = cs.Gray
		}

		tp.AddField(prNum)
		tp.AddField(pr.Title, tableprinter.WithTruncate(nil))
		tp.AddField(pr.HeadBranch())
		tp.AddField(pr.Author.DisplayName)
		tp.AddField(stateColor(pr.StateDisplay()))
		tp.EndRow()
	}

	return tp.Render()
}

func relativeTimeStr(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	return t.Format("Jan 2, 2006")
}
