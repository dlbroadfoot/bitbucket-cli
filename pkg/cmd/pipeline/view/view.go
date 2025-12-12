package view

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ViewOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser

	SelectorArg string
	Web         bool
	Steps       bool
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
	}

	cmd := &cobra.Command{
		Use:   "view <build-number>",
		Short: "View a pipeline",
		Long: heredoc.Doc(`
			View details of a pipeline run.

			With --web, open the pipeline in a web browser instead.
			With --steps, show the steps in the pipeline.
		`),
		Example: heredoc.Doc(`
			$ bb pipeline view 123
			$ bb pipeline view 123 --web
			$ bb pipeline view 123 --steps
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open pipeline in the browser")
	cmd.Flags().BoolVarP(&opts.Steps, "steps", "s", false, "Show pipeline steps")

	return cmd
}

func viewRun(opts *ViewOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the pipeline argument
	buildNumber, pipelineRepo, err := shared.ParsePipelineArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if pipelineRepo != nil {
		repo = pipelineRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	opts.IO.StartProgressIndicator()
	pipeline, err := fetchPipeline(httpClient, repo, buildNumber)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	openURL := pipeline.HTMLURL()

	if opts.Web {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	if err := printPipeline(opts.IO, pipeline); err != nil {
		return err
	}

	if opts.Steps {
		opts.IO.StartProgressIndicator()
		steps, err := fetchPipelineSteps(httpClient, repo, pipeline.UUID)
		opts.IO.StopProgressIndicator()

		if err != nil {
			return err
		}

		printSteps(opts.IO, steps)
	}

	return nil
}

func fetchPipeline(client *http.Client, repo bbrepo.Interface, buildNumber int) (*shared.Pipeline, error) {
	apiClient := api.NewClientFromHTTP(client)

	// First, get the pipeline UUID from the build number
	path := fmt.Sprintf("repositories/%s/%s/pipelines?q=build_number=%d",
		repo.RepoWorkspace(), repo.RepoSlug(), buildNumber)

	var result shared.PipelineList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Values) == 0 {
		return nil, fmt.Errorf("pipeline #%d not found", buildNumber)
	}

	return &result.Values[0], nil
}

func fetchPipelineSteps(client *http.Client, repo bbrepo.Interface, pipelineUUID string) ([]shared.Step, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps",
		repo.RepoWorkspace(), repo.RepoSlug(), pipelineUUID)

	var result shared.StepList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printPipeline(io *iostreams.IOStreams, pipeline *shared.Pipeline) error {
	cs := io.ColorScheme()
	out := io.Out

	// Header
	fmt.Fprintf(out, "%s #%d\n", cs.Bold("Pipeline"), pipeline.BuildNumber)
	fmt.Fprintln(out)

	// Status with color
	status := pipeline.StatusString()
	var statusColor func(string) string
	if pipeline.State != nil {
		switch pipeline.State.Name {
		case "COMPLETED":
			if pipeline.State.Result != nil && pipeline.State.Result.Name == "SUCCESSFUL" {
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
	fmt.Fprintf(out, "%s %s\n", cs.Bold("Status:"), statusColor(status))

	// Branch/Ref
	if pipeline.Target != nil && pipeline.Target.RefName != "" {
		fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Branch:"), pipeline.Target.RefName, pipeline.Target.RefType)
	}

	// Commit
	if pipeline.Target != nil && pipeline.Target.Commit != nil {
		hash := pipeline.Target.Commit.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Commit:"), hash)
		if pipeline.Target.Commit.Message != "" {
			// First line of commit message
			msg := pipeline.Target.Commit.Message
			if idx := len(msg); idx > 60 {
				msg = msg[:57] + "..."
			}
			fmt.Fprintf(out, "        %s\n", cs.Gray(msg))
		}
	}

	// Pull Request
	if pipeline.Target != nil && pipeline.Target.PullRequest != nil {
		fmt.Fprintf(out, "%s #%d\n", cs.Bold("Pull Request:"), pipeline.Target.PullRequest.ID)
	}

	// Creator
	if pipeline.Creator != nil {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Creator:"), pipeline.Creator.DisplayName)
	}

	// Trigger
	if pipeline.Trigger != nil {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Trigger:"), pipeline.Trigger.Name)
	}

	// Duration
	if pipeline.DurationIn > 0 {
		duration := time.Duration(pipeline.DurationIn) * time.Second
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Duration:"), duration.String())
	}

	// Created
	if t, err := time.Parse(time.RFC3339, pipeline.CreatedOn); err == nil {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Created:"), text.FuzzyAgo(time.Now(), t))
	}

	// URL
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", cs.Gray(pipeline.HTMLURL()))

	return nil
}

func printSteps(io *iostreams.IOStreams, steps []shared.Step) {
	cs := io.ColorScheme()
	out := io.Out

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", cs.Bold("── Steps ──"))
	fmt.Fprintln(out)

	if len(steps) == 0 {
		fmt.Fprintf(out, "%s\n", cs.Gray("No steps found"))
		return
	}

	for i, step := range steps {
		// Status indicator
		var statusIcon string
		var statusColor func(string) string
		if step.State != nil {
			switch step.State.Name {
			case "COMPLETED":
				if step.State.Result != nil && step.State.Result.Name == "SUCCESSFUL" {
					statusIcon = "✓"
					statusColor = cs.Green
				} else {
					statusIcon = "✗"
					statusColor = cs.Red
				}
			case "IN_PROGRESS":
				statusIcon = "●"
				statusColor = cs.Yellow
			case "PENDING":
				statusIcon = "○"
				statusColor = cs.Gray
			default:
				statusIcon = "○"
				statusColor = cs.Gray
			}
		} else {
			statusIcon = "○"
			statusColor = cs.Gray
		}

		// Step name
		name := step.Name
		if name == "" {
			name = fmt.Sprintf("Step %d", i+1)
		}

		// Duration
		var duration string
		if step.DurationIn > 0 {
			duration = fmt.Sprintf(" (%s)", time.Duration(step.DurationIn)*time.Second)
		}

		fmt.Fprintf(out, "  %s %s%s\n", statusColor(statusIcon), name, cs.Gray(duration))

		// Image
		if step.Image != nil && step.Image.Name != "" {
			fmt.Fprintf(out, "    %s\n", cs.Gray("Image: "+step.Image.Name))
		}
	}
}
