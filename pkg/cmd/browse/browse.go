package browse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type BrowseOptions struct {
	BaseRepo  func() (bbrepo.Interface, error)
	Browser   browser.Browser
	IO        *iostreams.IOStreams

	SelectorArg   string
	Branch        string
	SettingsFlag  bool
	WikiFlag      bool
	PipelinesFlag bool
	PRsFlag       bool
	IssuesFlag    bool
	NoBrowserFlag bool
}

func NewCmdBrowse(f *cmdutil.Factory, runF func(*BrowseOptions) error) *cobra.Command {
	opts := &BrowseOptions{
		Browser: f.Browser,
		IO:      f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "browse [<number> | <path>]",
		Short: "Open the repository in the browser",
		Long: heredoc.Doc(`
			Open the repository in the browser.

			With no arguments, the repository home page is opened.
			With a number argument, open that issue or pull request.
			With a path argument, open that file or directory.
		`),
		Example: heredoc.Doc(`
			# Open the repository home page
			$ bb browse

			# Open an issue or pull request by number
			$ bb browse 123

			# Open repository settings
			$ bb browse --settings

			# Open repository pipelines
			$ bb browse --pipelines

			# Open a specific file
			$ bb browse src/main.go

			# Open a file at a specific branch
			$ bb browse src/main.go --branch develop

			# Print URL without opening browser
			$ bb browse --no-browser
		`),
		Args:    cobra.MaximumNArgs(1),
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BaseRepo = f.BaseRepo

			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return runBrowse(opts)
		},
	}

	cmdutil.EnableRepoOverride(cmd, f)
	cmd.Flags().BoolVarP(&opts.SettingsFlag, "settings", "s", false, "Open repository settings")
	cmd.Flags().BoolVarP(&opts.WikiFlag, "wiki", "w", false, "Open repository wiki")
	cmd.Flags().BoolVarP(&opts.PipelinesFlag, "pipelines", "p", false, "Open repository pipelines")
	cmd.Flags().BoolVar(&opts.PRsFlag, "prs", false, "Open repository pull requests")
	cmd.Flags().BoolVar(&opts.IssuesFlag, "issues", false, "Open repository issues")
	cmd.Flags().BoolVarP(&opts.NoBrowserFlag, "no-browser", "n", false, "Print destination URL instead of opening the browser")
	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Select another branch by passing in the branch name")

	return cmd
}

func runBrowse(opts *BrowseOptions) error {
	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return fmt.Errorf("unable to determine base repository: %w", err)
	}

	section := parseSection(opts)
	url := generateRepoURL(baseRepo, section)

	if opts.NoBrowserFlag {
		_, err = fmt.Fprintln(opts.IO.Out, url)
		return err
	}

	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.Out, "Opening %s in your browser.\n", text.DisplayURL(url))
	}
	return opts.Browser.Browse(url)
}

func parseSection(opts *BrowseOptions) string {
	if opts.SettingsFlag {
		return "admin"
	} else if opts.WikiFlag {
		return "wiki"
	} else if opts.PipelinesFlag {
		return "pipelines"
	} else if opts.PRsFlag {
		return "pull-requests"
	} else if opts.IssuesFlag {
		return "issues"
	}

	if opts.SelectorArg == "" {
		return ""
	}

	// Check if it's a number (issue or PR)
	if isNumber(opts.SelectorArg) {
		// In Bitbucket, we need to determine if it's an issue or PR
		// Default to pull-requests since that's more common
		return fmt.Sprintf("pull-requests/%s", strings.TrimPrefix(opts.SelectorArg, "#"))
	}

	// It's a file path
	branch := opts.Branch
	if branch == "" {
		branch = "master" // Default branch - ideally we'd fetch this
	}

	return fmt.Sprintf("src/%s/%s", branch, opts.SelectorArg)
}

func generateRepoURL(repo bbrepo.Interface, section string) string {
	baseURL := fmt.Sprintf("https://%s/%s/%s", repo.RepoHost(), repo.RepoWorkspace(), repo.RepoSlug())
	if section == "" {
		return baseURL
	}
	return fmt.Sprintf("%s/%s", baseURL, section)
}

func isNumber(arg string) bool {
	_, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	return err == nil
}

// sha1 and sha256 are supported
var commitHash = regexp.MustCompile(`\A[a-f0-9]{7,64}\z`)

func isCommit(arg string) bool {
	return commitHash.MatchString(arg)
}
