package cancel

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

type CancelOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
}

func NewCmdCancel(f *cmdutil.Factory, runF func(*CancelOptions) error) *cobra.Command {
	opts := &CancelOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "cancel <build-number>",
		Short: "Cancel a running pipeline",
		Long: heredoc.Doc(`
			Cancel a running pipeline.

			This stops the pipeline and marks it as stopped.
		`),
		Example: heredoc.Doc(`
			$ bb pipeline cancel 123
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return cancelRun(opts)
		},
	}

	return cmd
}

func cancelRun(opts *CancelOptions) error {
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

	// First, fetch the pipeline to get its UUID and check if it's running
	opts.IO.StartProgressIndicator()
	pipeline, err := fetchPipeline(httpClient, repo, buildNumber)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return err
	}

	if !pipeline.IsRunning() {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("pipeline #%d is not running (status: %s)", buildNumber, pipeline.StatusString())
	}

	// Stop the pipeline
	err = stopPipeline(httpClient, repo, pipeline.UUID)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Cancelled pipeline #%d\n", cs.SuccessIcon(), buildNumber)
	}

	return nil
}

func fetchPipeline(client *http.Client, repo bbrepo.Interface, buildNumber int) (*shared.Pipeline, error) {
	apiClient := api.NewClientFromHTTP(client)

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

func stopPipeline(client *http.Client, repo bbrepo.Interface, pipelineUUID string) error {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/pipelines/%s/stopPipeline",
		repo.RepoWorkspace(), repo.RepoSlug(), pipelineUUID)

	return apiClient.Post(repo.RepoHost(), path, nil, nil)
}
