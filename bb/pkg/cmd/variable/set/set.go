package set

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type SetOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	VariableName  string
	VariableValue string
	Environment   string
	Body          string
}

func NewCmdSet(f *cmdutil.Factory, runF func(*SetOptions) error) *cobra.Command {
	opts := &SetOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "set <variable-name> [<value>]",
		Short: "Create or update a pipeline variable for a repository",
		Long: heredoc.Doc(`
			Create or update a pipeline variable for a repository.

			The variable value can be provided as:
			- A second argument
			- The --body flag
			- Standard input (e.g., piped from another command)

			Variables are stored as plain text. For sensitive data like API keys
			and passwords, use 'bb secret set' instead.
		`),
		Example: heredoc.Doc(`
			# Set a variable with value as argument
			$ bb variable set NODE_ENV production

			# Set a variable from a value
			$ bb variable set BUILD_FLAGS --body "-O2 -Wall"

			# Set a variable from stdin
			$ echo "value" | bb variable set MY_VAR

			# Set a variable for a specific environment
			$ bb variable set NODE_ENV production --environment staging
		`),
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.VariableName = args[0]
			if len(args) > 1 {
				opts.VariableValue = args[1]
			}

			if runF != nil {
				return runF(opts)
			}
			return setRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Variable value (alternative to positional argument)")
	cmd.Flags().StringVarP(&opts.Environment, "environment", "e", "", "Set variable for a specific deployment environment")

	return cmd
}

func setRun(opts *SetOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	// Get the variable value
	var value string
	if opts.VariableValue != "" {
		value = opts.VariableValue
	} else if opts.Body != "" {
		value = opts.Body
	} else if !opts.IO.IsStdinTTY() {
		// Read from stdin
		data, err := io.ReadAll(opts.IO.In)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		value = strings.TrimSpace(string(data))
	} else {
		return cmdutil.FlagErrorf("variable value is required")
	}

	opts.IO.StartProgressIndicator()
	err = createOrUpdateVariable(httpClient, repo, opts.VariableName, value, opts.Environment)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		if opts.Environment != "" {
			fmt.Fprintf(opts.IO.Out, "%s Set variable %s for environment %s\n",
				cs.SuccessIcon(), cs.Bold(opts.VariableName), cs.Cyan(opts.Environment))
		} else {
			fmt.Fprintf(opts.IO.Out, "%s Set variable %s\n",
				cs.SuccessIcon(), cs.Bold(opts.VariableName))
		}
	}

	return nil
}

type variablePayload struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

func createOrUpdateVariable(client *http.Client, repo bbrepo.Interface, name, value, environment string) error {
	apiClient := api.NewClientFromHTTP(client)

	var path string
	if environment != "" {
		path = fmt.Sprintf("repositories/%s/%s/deployments_config/environments/%s/variables",
			repo.RepoWorkspace(), repo.RepoSlug(), environment)
	} else {
		path = fmt.Sprintf("repositories/%s/%s/pipelines_config/variables",
			repo.RepoWorkspace(), repo.RepoSlug())
	}

	payload := variablePayload{
		Key:     name,
		Value:   value,
		Secured: false, // Variables are not secured; use secrets for that
	}

	// Try to create first, if it fails with conflict, try to update
	err := apiClient.Post(repo.RepoHost(), path, payload, nil)
	if err != nil {
		// Check if it's a conflict (variable already exists)
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "already exists") {
			// Try to update by finding and updating the existing variable
			return updateExistingVariable(apiClient, repo, name, value, environment)
		}
		return err
	}

	return nil
}

func updateExistingVariable(apiClient *api.Client, repo bbrepo.Interface, name, value, environment string) error {
	// First, find the existing variable's UUID
	var listPath string
	if environment != "" {
		listPath = fmt.Sprintf("repositories/%s/%s/deployments_config/environments/%s/variables?pagelen=100",
			repo.RepoWorkspace(), repo.RepoSlug(), environment)
	} else {
		listPath = fmt.Sprintf("repositories/%s/%s/pipelines_config/variables?pagelen=100",
			repo.RepoWorkspace(), repo.RepoSlug())
	}

	type VariableList struct {
		Values []struct {
			UUID string `json:"uuid"`
			Key  string `json:"key"`
		} `json:"values"`
	}

	var list VariableList
	if err := apiClient.Get(repo.RepoHost(), listPath, &list); err != nil {
		return err
	}

	var uuid string
	for _, v := range list.Values {
		if v.Key == name {
			uuid = v.UUID
			break
		}
	}

	if uuid == "" {
		return fmt.Errorf("variable %q not found", name)
	}

	// Update the variable
	var updatePath string
	// Remove braces from UUID if present
	uuid = strings.Trim(uuid, "{}")
	if environment != "" {
		updatePath = fmt.Sprintf("repositories/%s/%s/deployments_config/environments/%s/variables/{%s}",
			repo.RepoWorkspace(), repo.RepoSlug(), environment, uuid)
	} else {
		updatePath = fmt.Sprintf("repositories/%s/%s/pipelines_config/variables/{%s}",
			repo.RepoWorkspace(), repo.RepoSlug(), uuid)
	}

	payload := variablePayload{
		Key:     name,
		Value:   value,
		Secured: false,
	}

	return apiClient.Put(repo.RepoHost(), updatePath, payload, nil)
}
