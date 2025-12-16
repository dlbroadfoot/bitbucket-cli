package list

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	IO         *iostreams.IOStreams
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	BaseRepo   func() (bbrepo.Interface, error)

	State    string
	Kind     string
	Priority string
	Assignee string
	Reporter string
	Limit    int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Config:     f.Config,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues in a repository",
		Long: heredoc.Doc(`
			List issues in a Bitbucket repository.

			By default, this will list open issues.
		`),
		Example: heredoc.Doc(`
			# List open issues
			$ bb issue list

			# List all issues
			$ bb issue list --state all

			# List bugs only
			$ bb issue list --kind bug

			# List critical issues
			$ bb issue list --priority critical
		`),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "open", "Filter by state: {new|open|resolved|on hold|invalid|duplicate|wontfix|closed|all}")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "", "Filter by kind: {bug|enhancement|proposal|task}")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "", "Filter by priority: {trivial|minor|major|critical|blocker}")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Filter by assignee")
	cmd.Flags().StringVar(&opts.Reporter, "reporter", "", "Filter by reporter")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of issues to fetch")

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

	issues, err := fetchIssues(httpClient, repo, opts)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		fmt.Fprintln(opts.IO.ErrOut, "No issues match your search")
		return nil
	}

	return printIssues(opts.IO, issues)
}

func printIssues(io *iostreams.IOStreams, issues []shared.Issue) error {
	cs := io.ColorScheme()
	tp := tableprinter.New(io, tableprinter.WithHeader("ID", "TITLE", "STATE", "KIND", "PRIORITY", "REPORTER", "UPDATED"))

	for _, issue := range issues {
		// Color state
		stateColor := cs.ColorFromString("white")
		switch issue.State {
		case "new", "open":
			stateColor = cs.ColorFromString("green")
		case "resolved", "closed":
			stateColor = cs.ColorFromString("magenta")
		case "on hold":
			stateColor = cs.ColorFromString("yellow")
		case "invalid", "duplicate", "wontfix":
			stateColor = cs.ColorFromString("red")
		}

		// Color priority
		priorityColor := cs.ColorFromString("white")
		switch issue.Priority {
		case "blocker", "critical":
			priorityColor = cs.ColorFromString("red")
		case "major":
			priorityColor = cs.ColorFromString("yellow")
		}

		// Format updated time
		updatedAt := issue.UpdatedOn
		if t, err := time.Parse(time.RFC3339, issue.UpdatedOn); err == nil {
			updatedAt = t.Format("2006-01-02")
		}

		tp.AddField(fmt.Sprintf("#%d", issue.ID))
		tp.AddField(issue.Title, tableprinter.WithTruncate(nil))
		tp.AddField(issue.StateDisplay(), tableprinter.WithColor(stateColor))
		tp.AddField(issue.KindDisplay())
		tp.AddField(issue.PriorityDisplay(), tableprinter.WithColor(priorityColor))
		tp.AddField(issue.Reporter.DisplayName)
		tp.AddField(updatedAt)
		tp.EndRow()
	}

	return tp.Render()
}
