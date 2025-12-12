package pr

import (
	"github.com/MakeNowJust/heredoc"
	checkoutCmd "github.com/cli/bb/v2/pkg/cmd/pr/checkout"
	createCmd "github.com/cli/bb/v2/pkg/cmd/pr/create"
	listCmd "github.com/cli/bb/v2/pkg/cmd/pr/list"
	mergeCmd "github.com/cli/bb/v2/pkg/cmd/pr/merge"
	viewCmd "github.com/cli/bb/v2/pkg/cmd/pr/view"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdPR(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr <command>",
		Short: "Manage pull requests",
		Long:  "Work with Bitbucket pull requests.",
		Example: heredoc.Doc(`
			$ bb pr list
			$ bb pr view 123
			$ bb pr create
			$ bb pr merge 123
		`),
		Annotations: map[string]string{
			"help:arguments": heredoc.Doc(`
				A pull request can be supplied as argument in any of the following formats:
				- by number, e.g. "123"
				- by URL, e.g. "https://bitbucket.org/WORKSPACE/REPO/pull-requests/123"
			`),
		},
		GroupID: "core",
	}

	cmdutil.EnableRepoOverride(cmd, f)

	cmd.AddCommand(listCmd.NewCmdList(f, nil))
	cmd.AddCommand(viewCmd.NewCmdView(f, nil))
	cmd.AddCommand(createCmd.NewCmdCreate(f, nil))
	cmd.AddCommand(mergeCmd.NewCmdMerge(f, nil))
	cmd.AddCommand(checkoutCmd.NewCmdCheckout(f, nil))

	return cmd
}
