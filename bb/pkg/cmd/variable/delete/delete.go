package delete

import (
	"fmt"
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

type DeleteOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Prompter   prompter.Prompter

	VariableName string
	Environment  string
	Confirm      bool
}

func NewCmdDelete(f *cmdutil.Factory, runF func(*DeleteOptions) error) *cobra.Command {
	opts := &DeleteOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Prompter:   f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "delete <variable-name>",
		Short: "Delete a pipeline variable from a repository",
		Long: heredoc.Doc(`
			Delete a pipeline variable from a repository.

			By default, deletes repository-level variables. Use --environment to delete
			a variable from a specific deployment environment.
		`),
		Example: heredoc.Doc(`
			$ bb variable delete NODE_ENV
			$ bb variable delete NODE_ENV --yes
			$ bb variable delete NODE_ENV --environment production
		`),
		Aliases: []string{"remove", "rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.VariableName = args[0]

			if runF != nil {
				return runF(opts)
			}
			return deleteRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Environment, "environment", "e", "", "Delete variable from a specific deployment environment")
	cmd.Flags().BoolVarP(&opts.Confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func deleteRun(opts *DeleteOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	// Confirm deletion
	if !opts.Confirm {
		if !opts.IO.CanPrompt() {
			return cmdutil.FlagErrorf("--yes required when not running interactively")
		}

		var msg string
		if opts.Environment != "" {
			msg = fmt.Sprintf("Are you sure you want to delete variable %q from environment %q?",
				opts.VariableName, opts.Environment)
		} else {
			msg = fmt.Sprintf("Are you sure you want to delete variable %q?", opts.VariableName)
		}

		confirmed, err := opts.Prompter.Confirm(msg, false)
		if err != nil {
			return err
		}
		if !confirmed {
			return cmdutil.CancelError
		}
	}

	opts.IO.StartProgressIndicator()
	err = deleteVariable(httpClient, repo, opts.VariableName, opts.Environment)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		if opts.Environment != "" {
			fmt.Fprintf(opts.IO.Out, "%s Deleted variable %s from environment %s\n",
				cs.SuccessIcon(), cs.Bold(opts.VariableName), cs.Cyan(opts.Environment))
		} else {
			fmt.Fprintf(opts.IO.Out, "%s Deleted variable %s\n",
				cs.SuccessIcon(), cs.Bold(opts.VariableName))
		}
	}

	return nil
}

func deleteVariable(client *http.Client, repo bbrepo.Interface, name, environment string) error {
	apiClient := api.NewClientFromHTTP(client)

	// First, find the variable's UUID
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

	// Delete the variable
	// Remove braces from UUID if present
	uuid = strings.Trim(uuid, "{}")
	var deletePath string
	if environment != "" {
		deletePath = fmt.Sprintf("repositories/%s/%s/deployments_config/environments/%s/variables/{%s}",
			repo.RepoWorkspace(), repo.RepoSlug(), environment, uuid)
	} else {
		deletePath = fmt.Sprintf("repositories/%s/%s/pipelines_config/variables/{%s}",
			repo.RepoWorkspace(), repo.RepoSlug(), uuid)
	}

	return apiClient.Delete(repo.RepoHost(), deletePath)
}
