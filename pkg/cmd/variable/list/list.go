package list

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	Environment string
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline variables for a repository",
		Long: heredoc.Doc(`
			List pipeline variables for a repository.

			By default, lists repository-level variables. Use --environment to list
			variables for a specific deployment environment.

			This shows all variables including both plain and secured ones.
			For secured variables, the value is not displayed.
		`),
		Example: heredoc.Doc(`
			$ bb variable list
			$ bb variable list --environment production
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

	cmd.Flags().StringVarP(&opts.Environment, "environment", "e", "", "List variables for a specific deployment environment")

	return cmd
}

// PipelineVariable represents a Bitbucket pipeline variable
type PipelineVariable struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// PipelineVariableList represents a paginated list of pipeline variables
type PipelineVariableList struct {
	Size     int                `json:"size"`
	Page     int                `json:"page"`
	PageLen  int                `json:"pagelen"`
	Next     string             `json:"next"`
	Previous string             `json:"previous"`
	Values   []PipelineVariable `json:"values"`
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
	variables, err := fetchVariables(httpClient, repo, opts.Environment)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(variables) == 0 {
		if opts.Environment != "" {
			fmt.Fprintf(opts.IO.Out, "No variables found for environment %q\n", opts.Environment)
		} else {
			fmt.Fprintln(opts.IO.Out, "No variables found for this repository")
		}
		return nil
	}

	return printVariables(opts.IO, variables)
}

func fetchVariables(client *http.Client, repo bbrepo.Interface, environment string) ([]PipelineVariable, error) {
	apiClient := api.NewClientFromHTTP(client)

	var path string
	if environment != "" {
		path = fmt.Sprintf("repositories/%s/%s/deployments_config/environments/%s/variables?pagelen=100",
			repo.RepoWorkspace(), repo.RepoSlug(), environment)
	} else {
		path = fmt.Sprintf("repositories/%s/%s/pipelines_config/variables?pagelen=100",
			repo.RepoWorkspace(), repo.RepoSlug())
	}

	var result PipelineVariableList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printVariables(io *iostreams.IOStreams, variables []PipelineVariable) error {
	cs := io.ColorScheme()

	tp := tableprinter.New(io, tableprinter.WithHeader("NAME", "VALUE", "TYPE"))

	for _, v := range variables {
		tp.AddField(v.Key)

		if v.Secured {
			tp.AddField(cs.Gray("********"))
			tp.AddField(cs.Yellow("secured"))
		} else {
			// For non-secured variables, show truncated value
			value := v.Value
			if len(value) > 40 {
				value = value[:37] + "..."
			}
			tp.AddField(value)
			tp.AddField("plain")
		}

		tp.EndRow()
	}

	return tp.Render()
}
