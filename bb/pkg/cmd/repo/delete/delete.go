package delete

import (
	"fmt"
	"net/http"

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

	RepoArg string
	Confirm bool
}

func NewCmdDelete(f *cmdutil.Factory, runF func(*DeleteOptions) error) *cobra.Command {
	opts := &DeleteOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Prompter:   f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "delete [<repository>]",
		Short: "Delete a repository",
		Long: heredoc.Doc(`
			Delete a repository.

			Without an argument, deletes the current repository.

			This action is irreversible. All repository data, including issues,
			pull requests, and pipelines will be permanently deleted.
		`),
		Example: heredoc.Doc(`
			$ bb repo delete
			$ bb repo delete myworkspace/myrepo
			$ bb repo delete myworkspace/myrepo --yes
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RepoArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return deleteRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func deleteRun(opts *DeleteOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	var repo bbrepo.Interface
	if opts.RepoArg != "" {
		repo, err = bbrepo.FromFullName(opts.RepoArg)
		if err != nil {
			return err
		}
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	fullName := fmt.Sprintf("%s/%s", repo.RepoWorkspace(), repo.RepoSlug())

	// Confirm deletion
	if !opts.Confirm {
		if !opts.IO.CanPrompt() {
			return cmdutil.FlagErrorf("--yes required when not running interactively")
		}

		fmt.Fprintf(opts.IO.ErrOut, "You are about to delete the repository %s.\nThis action CANNOT be undone.\n", fullName)
		err = opts.Prompter.ConfirmDeletion(fullName)
		if err != nil {
			return err
		}
	}

	opts.IO.StartProgressIndicator()
	err = deleteRepository(httpClient, repo)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Deleted repository %s\n", cs.SuccessIcon(), fullName)
	}

	return nil
}

func deleteRepository(client *http.Client, repo bbrepo.Interface) error {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s", repo.RepoWorkspace(), repo.RepoSlug())

	return apiClient.Delete(repo.RepoHost(), path)
}
