package set

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/prompter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type SetOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Prompter   prompter.Prompter

	SecretName  string
	SecretValue string
	Environment string
	Body        string
}

func NewCmdSet(f *cmdutil.Factory, runF func(*SetOptions) error) *cobra.Command {
	opts := &SetOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Prompter:   f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "set <secret-name>",
		Short: "Create or update a secret for a repository",
		Long: heredoc.Doc(`
			Create or update a pipeline secret (secured variable) for a repository.

			The secret value can be provided via:
			- The --body flag
			- Standard input (e.g., piped from another command)
			- Interactive prompt (if neither of the above)

			Secrets are always stored as secured variables (encrypted).
		`),
		Example: heredoc.Doc(`
			# Set a secret interactively
			$ bb secret set API_KEY

			# Set a secret from a value
			$ bb secret set API_KEY --body "my-secret-value"

			# Set a secret from stdin
			$ echo "my-secret-value" | bb secret set API_KEY

			# Set a secret for a specific environment
			$ bb secret set API_KEY --environment production --body "prod-value"
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SecretName = args[0]

			if runF != nil {
				return runF(opts)
			}
			return setRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Secret value")
	cmd.Flags().StringVarP(&opts.Environment, "environment", "e", "", "Set secret for a specific deployment environment")

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

	// Get the secret value
	var secretValue string
	if opts.Body != "" {
		secretValue = opts.Body
	} else if !opts.IO.IsStdinTTY() {
		// Read from stdin
		data, err := io.ReadAll(opts.IO.In)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		secretValue = strings.TrimSpace(string(data))
	} else {
		// Interactive prompt
		if !opts.IO.CanPrompt() {
			return cmdutil.FlagErrorf("--body required when not running interactively")
		}
		secretValue, err = opts.Prompter.Password(fmt.Sprintf("Enter secret value for %s:", opts.SecretName))
		if err != nil {
			return err
		}
	}

	if secretValue == "" {
		return cmdutil.FlagErrorf("secret value cannot be empty")
	}

	opts.IO.StartProgressIndicator()
	err = createOrUpdateSecret(httpClient, repo, opts.SecretName, secretValue, opts.Environment)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		if opts.Environment != "" {
			fmt.Fprintf(opts.IO.Out, "%s Set secret %s for environment %s\n",
				cs.SuccessIcon(), cs.Bold(opts.SecretName), cs.Cyan(opts.Environment))
		} else {
			fmt.Fprintf(opts.IO.Out, "%s Set secret %s\n",
				cs.SuccessIcon(), cs.Bold(opts.SecretName))
		}
	}

	return nil
}

type secretPayload struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

func createOrUpdateSecret(client *http.Client, repo bbrepo.Interface, name, value, environment string) error {
	apiClient := api.NewClientFromHTTP(client)

	var path string
	if environment != "" {
		path = fmt.Sprintf("repositories/%s/%s/deployments_config/environments/%s/variables",
			repo.RepoWorkspace(), repo.RepoSlug(), environment)
	} else {
		path = fmt.Sprintf("repositories/%s/%s/pipelines_config/variables",
			repo.RepoWorkspace(), repo.RepoSlug())
	}

	payload := secretPayload{
		Key:     name,
		Value:   value,
		Secured: true,
	}

	// Try to create first, if it fails with conflict, try to update
	err := apiClient.Post(repo.RepoHost(), path, payload, nil)
	if err != nil {
		// Check if it's a conflict (variable already exists)
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "already exists") {
			// Try to update by finding and updating the existing variable
			return updateExistingSecret(apiClient, repo, name, value, environment)
		}
		return err
	}

	return nil
}

func updateExistingSecret(apiClient *api.Client, repo bbrepo.Interface, name, value, environment string) error {
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

	payload := secretPayload{
		Key:     name,
		Value:   value,
		Secured: true,
	}

	return apiClient.Put(repo.RepoHost(), updatePath, payload, nil)
}
