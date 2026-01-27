package run

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type RunOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	Branch    string
	Commit    string
	Pattern   string
	Custom    bool
	Variables map[string]string
}

func NewCmdRun(f *cmdutil.Factory, runF func(*RunOptions) error) *cobra.Command {
	opts := &RunOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Variables:  make(map[string]string),
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a pipeline run",
		Long: heredoc.Doc(`
			Trigger a new pipeline run.

			By default, runs the pipeline for the current branch.
			You can specify a branch with --branch or a specific commit with --commit.

			Use --pattern to run a specific pipeline configuration (e.g., for custom pipelines).
		`),
		Example: heredoc.Doc(`
			$ bb pipeline run
			$ bb pipeline run --branch main
			$ bb pipeline run --commit abc1234
			$ bb pipeline run --custom --pattern deploy
			$ bb pipeline run --variable KEY=value
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return runRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Branch to run pipeline for")
	cmd.Flags().StringVarP(&opts.Commit, "commit", "c", "", "Commit hash to run pipeline for")
	cmd.Flags().StringVarP(&opts.Pattern, "pattern", "p", "", "Pipeline pattern to run (for custom pipelines)")
	cmd.Flags().BoolVar(&opts.Custom, "custom", false, "Run a custom pipeline")
	cmd.Flags().StringToStringVar(&opts.Variables, "variable", nil, "Set pipeline variables (key=value)")

	return cmd
}

func runRun(opts *RunOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	// Default to main/master branch if not specified
	if opts.Branch == "" && opts.Commit == "" {
		opts.Branch = "main"
	}

	opts.IO.StartProgressIndicator()
	pipeline, err := triggerPipeline(httpClient, repo, opts)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Triggered pipeline #%d\n", cs.SuccessIcon(), pipeline.BuildNumber)
		fmt.Fprintln(opts.IO.Out, pipeline.HTMLURL())
	}

	return nil
}

type triggerPayload struct {
	Target    triggerTarget     `json:"target"`
	Variables []triggerVariable `json:"variables,omitempty"`
}

type triggerTarget struct {
	Type     string           `json:"type"`
	RefType  string           `json:"ref_type,omitempty"`
	RefName  string           `json:"ref_name,omitempty"`
	Commit   *triggerCommit   `json:"commit,omitempty"`
	Selector *triggerSelector `json:"selector,omitempty"`
}

type triggerCommit struct {
	Hash string `json:"hash"`
	Type string `json:"type"`
}

type triggerSelector struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

type triggerVariable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

func triggerPipeline(client *http.Client, repo bbrepo.Interface, opts *RunOptions) (*shared.Pipeline, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/pipelines",
		repo.RepoWorkspace(), repo.RepoSlug())

	payload := triggerPayload{
		Target: triggerTarget{
			Type: "pipeline_ref_target",
		},
	}

	if opts.Commit != "" {
		payload.Target.Type = "pipeline_commit_target"
		payload.Target.Commit = &triggerCommit{
			Hash: opts.Commit,
			Type: "commit",
		}
	} else {
		payload.Target.RefType = "branch"
		payload.Target.RefName = opts.Branch
	}

	// Custom pipeline
	if opts.Custom || opts.Pattern != "" {
		pattern := opts.Pattern
		if pattern == "" {
			pattern = "custom"
		}
		payload.Target.Selector = &triggerSelector{
			Type:    "custom",
			Pattern: pattern,
		}
	}

	// Variables
	for k, v := range opts.Variables {
		payload.Variables = append(payload.Variables, triggerVariable{
			Key:     k,
			Value:   v,
			Secured: false,
		})
	}

	var pipeline shared.Pipeline
	err := apiClient.Post(repo.RepoHost(), path, payload, &pipeline)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}
