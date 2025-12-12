package pr

import (
	"github.com/MakeNowJust/heredoc"
	checkoutCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/checkout"
	checksCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/checks"
	closeCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/close"
	commentCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/comment"
	createCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/create"
	diffCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/diff"
	editCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/edit"
	listCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/list"
	mergeCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/merge"
	reviewCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/review"
	statusCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/status"
	viewCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/view"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
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
	cmd.AddCommand(statusCmd.NewCmdStatus(f, nil))
	cmd.AddCommand(checksCmd.NewCmdChecks(f, nil))
	cmd.AddCommand(createCmd.NewCmdCreate(f, nil))
	cmd.AddCommand(editCmd.NewCmdEdit(f, nil))
	cmd.AddCommand(mergeCmd.NewCmdMerge(f, nil))
	cmd.AddCommand(checkoutCmd.NewCmdCheckout(f, nil))
	cmd.AddCommand(closeCmd.NewCmdClose(f, nil))
	cmd.AddCommand(commentCmd.NewCmdComment(f, nil))
	cmd.AddCommand(diffCmd.NewCmdDiff(f, nil))
	cmd.AddCommand(reviewCmd.NewCmdReview(f, nil))

	return cmd
}
