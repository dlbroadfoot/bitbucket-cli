package view

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/markdown"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	IO         *iostreams.IOStreams
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser

	IssueArg string
	Web      bool
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Config:     f.Config,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
	}

	cmd := &cobra.Command{
		Use:   "view {<number> | <url>}",
		Short: "View an issue",
		Long: heredoc.Doc(`
			Display the title, body, and other information about an issue.

			With '--web', open the issue in a web browser instead.
		`),
		Example: heredoc.Doc(`
			# View issue by number
			$ bb issue view 123

			# View issue in browser
			$ bb issue view 123 --web

			# View issue by URL
			$ bb issue view https://bitbucket.org/workspace/repo/issues/123
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IssueArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open the issue in the browser")

	return cmd
}

func viewRun(opts *ViewOptions) error {
	issueNum, issueRepo, err := shared.ParseIssueArg(opts.IssueArg)
	if err != nil {
		return err
	}

	var repo bbrepo.Interface
	if issueRepo != nil {
		repo = issueRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	issue, err := fetchIssue(httpClient, repo, issueNum)
	if err != nil {
		return err
	}

	if opts.Web {
		url := issue.HTMLURL()
		if url == "" {
			url = fmt.Sprintf("https://%s/%s/%s/issues/%d",
				repo.RepoHost(), repo.RepoWorkspace(), repo.RepoSlug(), issueNum)
		}
		return opts.Browser.Browse(url)
	}

	return printIssue(opts.IO, issue)
}

func fetchIssue(client *http.Client, repo bbrepo.Interface, issueNum int) (*shared.Issue, error) {
	apiURL := fmt.Sprintf("%srepositories/%s/%s/issues/%d",
		bbinstance.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		issueNum,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("issue #%d not found", issueNum)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var issue shared.Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func printIssue(io *iostreams.IOStreams, issue *shared.Issue) error {
	cs := io.ColorScheme()
	out := io.Out

	// Title and state
	fmt.Fprintf(out, "%s #%d\n", cs.Bold(issue.Title), issue.ID)

	// State with color
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
	fmt.Fprintf(out, "%s", stateColor(issue.StateDisplay()))

	// Priority
	if issue.Priority != "" {
		priorityColor := cs.ColorFromString("white")
		switch issue.Priority {
		case "blocker", "critical":
			priorityColor = cs.ColorFromString("red")
		case "major":
			priorityColor = cs.ColorFromString("yellow")
		}
		fmt.Fprintf(out, " • %s priority", priorityColor(issue.PriorityDisplay()))
	}

	// Kind
	if issue.Kind != "" {
		fmt.Fprintf(out, " • %s", issue.KindDisplay())
	}

	fmt.Fprintln(out)

	// Reporter and dates
	fmt.Fprintf(out, "%s opened this issue", issue.Reporter.DisplayName)
	if issue.CreatedOn != "" {
		if t, err := time.Parse(time.RFC3339, issue.CreatedOn); err == nil {
			fmt.Fprintf(out, " on %s", t.Format("Jan 2, 2006"))
		}
	}
	fmt.Fprintln(out)

	// Assignee
	if issue.Assignee != nil {
		fmt.Fprintf(out, "Assigned to: %s\n", issue.Assignee.DisplayName)
	}

	// Votes and watches
	if issue.Votes > 0 || issue.Watches > 0 {
		var stats []string
		if issue.Votes > 0 {
			stats = append(stats, fmt.Sprintf("%d votes", issue.Votes))
		}
		if issue.Watches > 0 {
			stats = append(stats, fmt.Sprintf("%d watchers", issue.Watches))
		}
		fmt.Fprintf(out, "%s\n", strings.Join(stats, " • "))
	}

	// Body
	body := issue.Body()
	if body != "" {
		fmt.Fprintln(out)
		if io.IsStdoutTTY() {
			rendered, err := markdown.Render(body,
				markdown.WithTheme(io.TerminalTheme()),
				markdown.WithWrap(io.TerminalWidth()))
			if err == nil {
				body = rendered
			}
		}
		fmt.Fprint(out, body)
	}

	// URL
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", cs.Gray(issue.HTMLURL()))

	return nil
}
