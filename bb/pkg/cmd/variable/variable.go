package variable

import (
	"github.com/MakeNowJust/heredoc"
	deleteCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/variable/delete"
	listCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/variable/list"
	setCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/variable/set"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdVariable(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variable <command>",
		Short: "Manage repository pipeline variables",
		Long: heredoc.Doc(`
			Manage repository pipeline variables for Bitbucket Pipelines.

			Variables are non-secured values that can be used in pipeline configurations.
			For sensitive data like API keys and passwords, use 'bb secret' instead.
		`),
		Example: heredoc.Doc(`
			$ bb variable list
			$ bb variable set NODE_ENV production
			$ bb variable delete NODE_ENV
		`),
		GroupID: "core",
	}

	cmd.AddCommand(listCmd.NewCmdList(f, nil))
	cmd.AddCommand(setCmd.NewCmdSet(f, nil))
	cmd.AddCommand(deleteCmd.NewCmdDelete(f, nil))

	return cmd
}
