package project

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/project/create"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/project/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/project/view"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdProject(f *cmdutil.Factory) *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "project <command>",
		Short: "Work with Bitbucket Projects.",
		Long: heredoc.Doc(`
			Work with Bitbucket Projects.

			Bitbucket Projects are used to organize repositories within a workspace.
		`),
		Example: heredoc.Doc(`
			$ bb project list --workspace myworkspace
			$ bb project view PROJ --workspace myworkspace
			$ bb project create "My Project" --key PROJ --workspace myworkspace
		`),
		GroupID: "core",
	}

	cmd.AddCommand(list.NewCmdList(f, nil))
	cmd.AddCommand(view.NewCmdView(f, nil))
	cmd.AddCommand(create.NewCmdCreate(f, nil))

	return cmd
}
