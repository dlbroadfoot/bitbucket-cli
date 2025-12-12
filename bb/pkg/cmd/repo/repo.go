package repo

import (
	"github.com/MakeNowJust/heredoc"
	repoCloneCmd "github.com/cli/bb/v2/pkg/cmd/repo/clone"
	repoListCmd "github.com/cli/bb/v2/pkg/cmd/repo/list"
	repoViewCmd "github.com/cli/bb/v2/pkg/cmd/repo/view"

	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo <command>",
		Short: "Manage repositories",
		Long:  `Work with Bitbucket repositories.`,
		Example: heredoc.Doc(`
			$ bb repo list
			$ bb repo clone workspace/repo
			$ bb repo view --web
		`),
		Annotations: map[string]string{
			"help:arguments": heredoc.Doc(`
				A repository can be supplied as an argument in any of the following formats:
				- "WORKSPACE/REPO_SLUG"
				- by URL, e.g. "https://bitbucket.org/WORKSPACE/REPO_SLUG"
			`),
		},
		GroupID: "core",
	}

	cmdutil.AddGroup(cmd, "General commands",
		repoListCmd.NewCmdList(f, nil),
	)

	cmdutil.AddGroup(cmd, "Targeted commands",
		repoViewCmd.NewCmdView(f, nil),
		repoCloneCmd.NewCmdClone(f, nil),
	)

	return cmd
}
