package project

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/pkg/cmdutil"
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
			$ bb project view myproject --workspace myworkspace
		`),
		GroupID: "core",
	}

	// TODO: Implement project subcommands for Bitbucket
	// - list: GET /workspaces/{workspace}/projects
	// - view: GET /workspaces/{workspace}/projects/{project_key}
	// - create: POST /workspaces/{workspace}/projects

	return cmd
}
