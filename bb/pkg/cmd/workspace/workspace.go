package workspace

import (
	"github.com/MakeNowJust/heredoc"
	listCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/workspace/list"
	viewCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/workspace/view"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdWorkspace(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace <command>",
		Short: "Manage Bitbucket workspaces",
		Long: heredoc.Doc(`
			Manage Bitbucket workspaces.

			Workspaces are the top-level organizational structure in Bitbucket.
			They contain projects and repositories.
		`),
		Example: heredoc.Doc(`
			$ bb workspace list
			$ bb workspace view myworkspace
		`),
		GroupID: "core",
	}

	cmd.AddCommand(listCmd.NewCmdList(f, nil))
	cmd.AddCommand(viewCmd.NewCmdView(f, nil))

	return cmd
}
