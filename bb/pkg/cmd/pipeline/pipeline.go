package pipeline

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/cancel"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/run"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pipeline/view"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdPipeline(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline <command>",
		Short: "Manage pipelines",
		Long:  "Work with Bitbucket Pipelines CI/CD.",
		Example: heredoc.Doc(`
			$ bb pipeline list
			$ bb pipeline view 123
			$ bb pipeline run --branch main
			$ bb pipeline cancel 123
		`),
		Aliases: []string{"pipelines"},
		GroupID: "core",
	}

	cmdutil.EnableRepoOverride(cmd, f)

	cmd.AddCommand(list.NewCmdList(f, nil))
	cmd.AddCommand(view.NewCmdView(f, nil))
	cmd.AddCommand(run.NewCmdRun(f, nil))
	cmd.AddCommand(cancel.NewCmdCancel(f, nil))

	return cmd
}
