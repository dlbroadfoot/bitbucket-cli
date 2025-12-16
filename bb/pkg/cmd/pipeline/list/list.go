package list

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	Branch string
	Status string
	Limit  int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List pipelines for a repository",
		Aliases: []string{"ls"},
		Long: heredoc.Doc(`
			List pipelines for a repository.

			By default, shows the most recent pipelines for all branches.
		`),
		Example: heredoc.Doc(`
			$ bb pipeline list
			$ bb pipeline list --branch main
			$ bb pipeline list --status successful
			$ bb pipeline list --limit 50
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Filter by branch name")
	cmd.Flags().StringVarP(&opts.Status, "status", "s", "", "Filter by status (pending, in_progress, successful, failed, stopped)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of pipelines to list")

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

	opts.IO.StartProgressIndicator()
	pipelines, err := fetchPipelines(httpClient, repo, opts)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(pipelines) == 0 {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintln(opts.IO.ErrOut, "No pipelines found")
		}
		return nil
	}

	return printPipelines(opts.IO, pipelines)
}

func fetchPipelines(client *http.Client, repo bbrepo.Interface, opts *ListOptions) ([]shared.Pipeline, error) {
	apiClient := api.NewClientFromHTTP(client)

	params := url.Values{}
	params.Set("sort", "-created_on")
	if opts.Limit > 0 {
		params.Set("pagelen", fmt.Sprintf("%d", opts.Limit))
	}

	// Build filter query
	var filters []string

	if opts.Branch != "" {
		filters = append(filters, fmt.Sprintf(`target.ref_name="%s"`, opts.Branch))
	}

	if opts.Status != "" {
		switch opts.Status {
		case "pending":
			filters = append(filters, `state.name="PENDING"`)
		case "in_progress", "running":
			filters = append(filters, `state.name="IN_PROGRESS"`)
		case "successful", "success":
			filters = append(filters, `state.name="COMPLETED" AND state.result.name="SUCCESSFUL"`)
		case "failed", "failure":
			filters = append(filters, `state.name="COMPLETED" AND state.result.name="FAILED"`)
		case "stopped":
			filters = append(filters, `state.name="COMPLETED" AND state.result.name="STOPPED"`)
		}
	}

	if len(filters) > 0 {
		q := ""
		for i, f := range filters {
			if i > 0 {
				q += " AND "
			}
			q += f
		}
		params.Set("q", q)
	}

	path := fmt.Sprintf("repositories/%s/%s/pipelines?%s",
		repo.RepoWorkspace(), repo.RepoSlug(), params.Encode())

	var result shared.PipelineList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printPipelines(io *iostreams.IOStreams, pipelines []shared.Pipeline) error {
	cs := io.ColorScheme()

	tp := tableprinter.New(io, tableprinter.WithHeader("#", "STATUS", "BRANCH", "COMMIT", "DURATION", "CREATED"))

	for _, p := range pipelines {
		tp.AddField(fmt.Sprintf("%d", p.BuildNumber))

		// Status with color
		status := p.StatusString()
		var statusColor func(string) string
		if p.State != nil {
			switch p.State.Name {
			case "COMPLETED":
				if p.State.Result != nil && p.State.Result.Name == "SUCCESSFUL" {
					statusColor = cs.Green
				} else {
					statusColor = cs.Red
				}
			case "IN_PROGRESS":
				statusColor = cs.Yellow
			case "PENDING":
				statusColor = cs.Gray
			default:
				statusColor = cs.Gray
			}
		} else {
			statusColor = cs.Gray
		}
		tp.AddField(statusColor(status))

		// Branch
		tp.AddField(p.RefName())

		// Commit
		tp.AddField(p.CommitHash())

		// Duration
		if p.DurationIn > 0 {
			duration := time.Duration(p.DurationIn) * time.Second
			tp.AddField(duration.String())
		} else {
			tp.AddField("-")
		}

		// Created
		if t, err := time.Parse(time.RFC3339, p.CreatedOn); err == nil {
			tp.AddField(text.FuzzyAgo(time.Now(), t))
		} else {
			tp.AddField("-")
		}

		tp.EndRow()
	}

	return tp.Render()
}
