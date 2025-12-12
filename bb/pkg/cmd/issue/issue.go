package issue

import (
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/close"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/comment"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/create"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/edit"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/reopen"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/view"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdIssue(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <command>",
		Short: "Manage issues",
		Long:  "Work with Bitbucket issues.",
		Example: `
$ bb issue list
$ bb issue view 123
$ bb issue create
`,
		GroupID: "core",
		Annotations: map[string]string{
			"help:arguments": `An issue can be supplied as argument in any of the following formats:
- by number, e.g. "123"
- by URL, e.g. "https://bitbucket.org/WORKSPACE/REPO/issues/123"`,
		},
	}

	cmdutil.EnableRepoOverride(cmd, f)

	cmd.AddCommand(list.NewCmdList(f, nil))
	cmd.AddCommand(view.NewCmdView(f, nil))
	cmd.AddCommand(create.NewCmdCreate(f, nil))
	cmd.AddCommand(edit.NewCmdEdit(f, nil))
	cmd.AddCommand(close.NewCmdClose(f, nil))
	cmd.AddCommand(reopen.NewCmdReopen(f, nil))
	cmd.AddCommand(comment.NewCmdComment(f, nil))

	return cmd
}
