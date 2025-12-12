package repo

import (
	"github.com/MakeNowJust/heredoc"
	repoCloneCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/clone"
	repoCreateCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/create"
	repoDeleteCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/delete"
	repoEditCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/edit"
	repoForkCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/fork"
	repoListCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/list"
	repoSyncCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/sync"
	repoViewCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/repo/view"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
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
		repoCreateCmd.NewCmdCreate(f, nil),
	)

	cmdutil.AddGroup(cmd, "Targeted commands",
		repoViewCmd.NewCmdView(f, nil),
		repoCloneCmd.NewCmdClone(f, nil),
		repoForkCmd.NewCmdFork(f, nil),
		repoEditCmd.NewCmdEdit(f, nil),
		repoDeleteCmd.NewCmdDelete(f, nil),
		repoSyncCmd.NewCmdSync(f, nil),
	)

	return cmd
}
