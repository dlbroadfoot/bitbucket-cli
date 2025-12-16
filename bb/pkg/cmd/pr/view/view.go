package view

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/markdown"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser

	SelectorArg string
	Web         bool
	Comments    bool
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
	}

	cmd := &cobra.Command{
		Use:   "view [<number> | <url>]",
		Short: "View a pull request",
		Long: heredoc.Doc(`
			Display the title, body, and other information about a pull request.

			Without an argument, the pull request that belongs to the current branch
			is displayed.

			With --web, open the pull request in a web browser instead.
		`),
		Example: heredoc.Doc(`
			# View pull request #123
			$ bb pr view 123

			# Open pull request #123 in browser
			$ bb pr view 123 --web

			# View pull request with comments
			$ bb pr view 123 --comments
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open pull request in the browser")
	cmd.Flags().BoolVarP(&opts.Comments, "comments", "c", false, "View pull request comments")

	return cmd
}

func viewRun(opts *ViewOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the PR argument first to check if it contains repo info
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

	pr, err := list.FetchPullRequest(httpClient, repo, prID)
	if err != nil {
		return err
	}

	openURL := pr.HTMLURL()

	if opts.Web {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	opts.IO.DetectTerminalTheme()
	if err := opts.IO.StartPager(); err == nil {
		defer opts.IO.StopPager()
	}

	if err := printPullRequest(opts.IO, pr); err != nil {
		return err
	}

	// Fetch and display comments if requested
	if opts.Comments && pr.CommentCount > 0 {
		comments, err := list.FetchPullRequestComments(httpClient, repo, prID)
		if err != nil {
			return err
		}
		printComments(opts.IO, comments)
	}

	return nil
}

func printPullRequest(io *iostreams.IOStreams, pr *shared.PullRequest) error {
	cs := io.ColorScheme()
	out := io.Out

	// Title and state
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

	fmt.Fprintf(out, "%s #%d\n", cs.Bold(pr.Title), pr.ID)
	fmt.Fprintf(out, "%s • %s wants to merge into %s from %s\n",
		stateColor(pr.StateDisplay()),
		pr.Author.DisplayName,
		cs.Cyan(pr.BaseBranch()),
		cs.Cyan(pr.HeadBranch()),
	)
	fmt.Fprintln(out)

	// Description
	if pr.Description != "" {
		description := pr.Description
		if io.IsStdoutTTY() {
			rendered, err := markdown.Render(description,
				markdown.WithTheme(io.TerminalTheme()),
				markdown.WithWrap(io.TerminalWidth()))
			if err == nil {
				description = rendered
			}
		}
		fmt.Fprintln(out, description)
	} else {
		fmt.Fprintf(out, "%s\n", cs.Gray("No description provided"))
	}
	fmt.Fprintln(out)

	// Reviewers
	if len(pr.Reviewers) > 0 {
		var reviewerNames []string
		for _, r := range pr.Reviewers {
			reviewerNames = append(reviewerNames, r.DisplayName)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Reviewers:"), strings.Join(reviewerNames, ", "))
	}

	// Stats
	if pr.CommentCount > 0 || pr.TaskCount > 0 {
		fmt.Fprintf(out, "%s %d  %s %d\n",
			cs.Bold("Comments:"), pr.CommentCount,
			cs.Bold("Tasks:"), pr.TaskCount)
	}

	// URL
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", cs.Gray(pr.HTMLURL()))

	return nil
}

func printComments(io *iostreams.IOStreams, comments []shared.Comment) {
	cs := io.ColorScheme()
	out := io.Out

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", cs.Bold("── Comments ──"))
	fmt.Fprintln(out)

	commentCount := 0
	for _, comment := range comments {
		// Skip deleted comments
		if comment.Deleted {
			continue
		}

		commentCount++

		// Parse and format the timestamp
		timestamp := comment.CreatedOn
		if t, err := time.Parse(time.RFC3339, comment.CreatedOn); err == nil {
			timestamp = text.FuzzyAgo(time.Now(), t)
		}

		// Author and timestamp with optional inline location
		if comment.Inline != nil {
			// This is an inline/code review comment
			location := comment.Inline.Path
			if comment.Inline.To != nil {
				location = fmt.Sprintf("%s:%d", comment.Inline.Path, *comment.Inline.To)
			}
			fmt.Fprintf(out, "%s %s %s\n",
				cs.Bold(comment.User.DisplayName),
				cs.Gray("commented "+timestamp),
				cs.Cyan("on "+location))
		} else {
			fmt.Fprintf(out, "%s %s\n", cs.Bold(comment.User.DisplayName), cs.Gray("commented "+timestamp))
		}

		// Comment content
		content := comment.Content.Raw
		if content != "" {
			if io.IsStdoutTTY() {
				rendered, err := markdown.Render(content,
					markdown.WithTheme(io.TerminalTheme()),
					markdown.WithWrap(io.TerminalWidth()))
				if err == nil {
					content = rendered
				}
			}
			fmt.Fprintln(out, content)
		}
		fmt.Fprintln(out)
	}

	if commentCount == 0 {
		fmt.Fprintf(out, "%s\n", cs.Gray("No comments"))
	}
}
