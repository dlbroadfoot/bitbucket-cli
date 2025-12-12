package delete

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/prompter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type DeleteOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	Prompter   prompter.Prompter

	KeyID   string
	Confirm bool
}

func NewCmdDelete(f *cmdutil.Factory, runF func(*DeleteOptions) error) *cobra.Command {
	opts := &DeleteOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Prompter:   f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Delete an SSH key from your Bitbucket account",
		Long: heredoc.Doc(`
			Delete an SSH key from your Bitbucket account.

			The key ID can be found using 'bb ssh-key list'.
		`),
		Example: heredoc.Doc(`
			$ bb ssh-key delete {abc123-def456}
			$ bb ssh-key delete {abc123-def456} --yes
		`),
		Aliases: []string{"remove", "rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.KeyID = args[0]

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

	// Confirm deletion
	if !opts.Confirm {
		if !opts.IO.CanPrompt() {
			return cmdutil.FlagErrorf("--yes required when not running interactively")
		}

		confirmed, err := opts.Prompter.Confirm(
			fmt.Sprintf("Are you sure you want to delete SSH key %s?", opts.KeyID), false)
		if err != nil {
			return err
		}
		if !confirmed {
			return cmdutil.CancelError
		}
	}

	opts.IO.StartProgressIndicator()
	err = deleteSSHKey(httpClient, opts.KeyID)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Deleted SSH key %s\n", cs.SuccessIcon(), opts.KeyID)
	}

	return nil
}

func deleteSSHKey(client *http.Client, keyID string) error {
	apiClient := api.NewClientFromHTTP(client)

	// Ensure keyID has braces
	if !strings.HasPrefix(keyID, "{") {
		keyID = "{" + keyID
	}
	if !strings.HasSuffix(keyID, "}") {
		keyID = keyID + "}"
	}

	path := fmt.Sprintf("user/ssh-keys/%s", keyID)

	return apiClient.Delete("bitbucket.org", path)
}
