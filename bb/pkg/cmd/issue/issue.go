package issue

import (
	"github.com/cli/bb/v2/pkg/cmd/issue/create"
	"github.com/cli/bb/v2/pkg/cmd/issue/list"
	"github.com/cli/bb/v2/pkg/cmd/issue/view"
	"github.com/cli/bb/v2/pkg/cmdutil"
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

	return cmd
}
