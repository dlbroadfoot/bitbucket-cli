package list

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	IO         *iostreams.IOStreams
	Exporter   cmdutil.Exporter

	Limit     int
	Workspace string

	Visibility string
	Fork       bool
	Source     bool
	Language   string

	Now func() time.Time
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := ListOptions{
		IO:         f.IOStreams,
		Config:     f.Config,
		HttpClient: f.HttpClient,
		Now:        time.Now,
	}

	var (
		flagPublic  bool
		flagPrivate bool
	)

	cmd := &cobra.Command{
		Use:   "list [<workspace>]",
		Args:  cobra.MaximumNArgs(1),
		Short: "List repositories in a workspace",
		Long: heredoc.Docf(`
			List repositories in a Bitbucket workspace.

			If no workspace is specified, lists repositories accessible to the authenticated user.

			Note that the list will only include repositories in the provided workspace,
			and the %[1]s--fork%[1]s or %[1]s--source%[1]s flags will not traverse ownership boundaries.
		`, "`"),
		Aliases: []string{"ls"},
		RunE: func(c *cobra.Command, args []string) error {
			if opts.Limit < 1 {
				return cmdutil.FlagErrorf("invalid limit: %v", opts.Limit)
			}

			if err := cmdutil.MutuallyExclusive("specify only one of `--public`, `--private`, or `--visibility`", flagPublic, flagPrivate, opts.Visibility != ""); err != nil {
				return err
			}
			if opts.Source && opts.Fork {
				return cmdutil.FlagErrorf("specify only one of `--source` or `--fork`")
			}

			if flagPrivate {
				opts.Visibility = "private"
			} else if flagPublic {
				opts.Visibility = "public"
			}

			if len(args) > 0 {
				opts.Workspace = args[0]
			}

			if runF != nil {
				return runF(&opts)
			}
			return listRun(&opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of repositories to list")
	cmd.Flags().BoolVar(&opts.Source, "source", false, "Show only non-forks")
	cmd.Flags().BoolVar(&opts.Fork, "fork", false, "Show only forks")
	cmd.Flags().StringVarP(&opts.Language, "language", "l", "", "Filter by primary coding language")
	cmdutil.StringEnumFlag(cmd, &opts.Visibility, "visibility", "", "", []string{"public", "private"}, "Filter by repository visibility")

	cmd.Flags().BoolVar(&flagPrivate, "private", false, "Show only private repositories")
	cmd.Flags().BoolVar(&flagPublic, "public", false, "Show only public repositories")
	_ = cmd.Flags().MarkDeprecated("public", "use `--visibility=public` instead")
	_ = cmd.Flags().MarkDeprecated("private", "use `--visibility=private` instead")

	return cmd
}

func listRun(opts *ListOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	// Get the default host, filtering out GitHub hosts from shared config
	host, _ := cfg.Authentication().DefaultHost()
	if strings.Contains(host, "github.com") || host == "" {
		host = bbinstance.Default()
	}

	filter := FilterOptions{
		Visibility: opts.Visibility,
		Fork:       opts.Fork,
		Source:     opts.Source,
		Language:   opts.Language,
	}

	listResult, err := listRepos(httpClient, host, opts.Limit, opts.Workspace, filter)
	if err != nil {
		return err
	}

	if opts.Workspace != "" && listResult.Owner == "" && !listResult.FromSearch {
		return fmt.Errorf("the workspace %q was not found", opts.Workspace)
	}

	if err := opts.IO.StartPager(); err != nil {
		fmt.Fprintf(opts.IO.ErrOut, "error starting pager: %v\n", err)
	}
	defer opts.IO.StopPager()

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, listResult.Repositories)
	}

	cs := opts.IO.ColorScheme()
	tp := tableprinter.New(opts.IO, tableprinter.WithHeader("NAME", "DESCRIPTION", "INFO", "UPDATED"))

	totalMatchCount := len(listResult.Repositories)
	for _, repo := range listResult.Repositories {
		info := repoInfo(repo)
		infoColor := cs.Muted

		if repo.IsPrivate {
			infoColor = cs.Yellow
		}

		t := repo.UpdatedOn
		fullName := repo.FullName
		if fullName == "" {
			fullName = fmt.Sprintf("%s/%s", repo.Workspace.Slug, repo.Slug)
		}

		tp.AddField(fullName, tableprinter.WithColor(cs.Bold))
		tp.AddField(text.RemoveExcessiveWhitespace(repo.Description))
		tp.AddField(info, tableprinter.WithColor(infoColor))
		tp.AddTimeField(opts.Now(), t, cs.Muted)
		tp.EndRow()
	}

	if opts.IO.IsStdoutTTY() {
		hasFilters := filter.Visibility != "" || filter.Fork || filter.Source || filter.Language != ""
		title := listHeader(listResult.Owner, totalMatchCount, listResult.TotalCount, hasFilters)
		fmt.Fprintf(opts.IO.Out, "\n%s\n\n", title)
	}

	if totalMatchCount > 0 {
		return tp.Render()
	}

	return nil
}

func listHeader(workspace string, matchCount, totalMatchCount int, hasFilters bool) string {
	if totalMatchCount == 0 {
		if hasFilters {
			return "No results match your search"
		} else if workspace != "" {
			return "There are no repositories in workspace " + workspace
		}
		return "No results"
	}

	var matchStr string
	if hasFilters {
		matchStr = " that match your search"
	}
	if workspace != "" {
		return fmt.Sprintf("Showing %d of %d repositories in workspace %s%s", matchCount, totalMatchCount, workspace, matchStr)
	}
	return fmt.Sprintf("Showing %d of %d repositories%s", matchCount, totalMatchCount, matchStr)
}

func repoInfo(r api.Repository) string {
	var tags []string

	if r.IsPrivate {
		tags = append(tags, "private")
	} else {
		tags = append(tags, "public")
	}

	if r.Parent != nil {
		tags = append(tags, "fork")
	}

	return strings.Join(tags, ", ")
}
