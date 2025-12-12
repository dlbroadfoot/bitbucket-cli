package secret

import (
	"github.com/MakeNowJust/heredoc"
	deleteCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/secret/delete"
	listCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/secret/list"
	setCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/secret/set"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdSecret(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret <command>",
		Short: "Manage repository secrets (secured pipeline variables)",
		Long: heredoc.Doc(`
			Manage repository secrets for Bitbucket Pipelines.

			Secrets are secured pipeline variables that are encrypted and cannot be read
			back after being set. They are commonly used for sensitive data like API keys,
			passwords, and tokens.
		`),
		Example: heredoc.Doc(`
			$ bb secret list
			$ bb secret set API_KEY
			$ bb secret delete API_KEY
		`),
		GroupID: "core",
	}

	cmd.AddCommand(listCmd.NewCmdList(f, nil))
	cmd.AddCommand(setCmd.NewCmdSet(f, nil))
	cmd.AddCommand(deleteCmd.NewCmdDelete(f, nil))

	return cmd
}
