package project

import (
	"github.com/MakeNowJust/heredoc"
	cmdClose "github.com/cli/bb/v2/pkg/cmd/project/close"
	cmdCopy "github.com/cli/bb/v2/pkg/cmd/project/copy"
	cmdCreate "github.com/cli/bb/v2/pkg/cmd/project/create"
	cmdDelete "github.com/cli/bb/v2/pkg/cmd/project/delete"
	cmdEdit "github.com/cli/bb/v2/pkg/cmd/project/edit"
	cmdFieldCreate "github.com/cli/bb/v2/pkg/cmd/project/field-create"
	cmdFieldDelete "github.com/cli/bb/v2/pkg/cmd/project/field-delete"
	cmdFieldList "github.com/cli/bb/v2/pkg/cmd/project/field-list"
	cmdItemAdd "github.com/cli/bb/v2/pkg/cmd/project/item-add"
	cmdItemArchive "github.com/cli/bb/v2/pkg/cmd/project/item-archive"
	cmdItemCreate "github.com/cli/bb/v2/pkg/cmd/project/item-create"
	cmdItemDelete "github.com/cli/bb/v2/pkg/cmd/project/item-delete"
	cmdItemEdit "github.com/cli/bb/v2/pkg/cmd/project/item-edit"
	cmdItemList "github.com/cli/bb/v2/pkg/cmd/project/item-list"
	cmdLink "github.com/cli/bb/v2/pkg/cmd/project/link"
	cmdList "github.com/cli/bb/v2/pkg/cmd/project/list"
	cmdTemplate "github.com/cli/bb/v2/pkg/cmd/project/mark-template"
	cmdUnlink "github.com/cli/bb/v2/pkg/cmd/project/unlink"
	cmdView "github.com/cli/bb/v2/pkg/cmd/project/view"
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
			$ bb project create --title "My Project" --workspace myworkspace
		`),
		GroupID: "core",
	}

	cmd.AddCommand(cmdList.NewCmdList(f, nil))
	cmd.AddCommand(cmdCreate.NewCmdCreate(f, nil))
	cmd.AddCommand(cmdCopy.NewCmdCopy(f, nil))
	cmd.AddCommand(cmdClose.NewCmdClose(f, nil))
	cmd.AddCommand(cmdDelete.NewCmdDelete(f, nil))
	cmd.AddCommand(cmdEdit.NewCmdEdit(f, nil))
	cmd.AddCommand(cmdLink.NewCmdLink(f, nil))
	cmd.AddCommand(cmdView.NewCmdView(f, nil))
	cmd.AddCommand(cmdTemplate.NewCmdMarkTemplate(f, nil))
	cmd.AddCommand(cmdUnlink.NewCmdUnlink(f, nil))

	// items
	cmd.AddCommand(cmdItemList.NewCmdList(f, nil))
	cmd.AddCommand(cmdItemCreate.NewCmdCreateItem(f, nil))
	cmd.AddCommand(cmdItemAdd.NewCmdAddItem(f, nil))
	cmd.AddCommand(cmdItemEdit.NewCmdEditItem(f, nil))
	cmd.AddCommand(cmdItemArchive.NewCmdArchiveItem(f, nil))
	cmd.AddCommand(cmdItemDelete.NewCmdDeleteItem(f, nil))

	// fields
	cmd.AddCommand(cmdFieldList.NewCmdList(f, nil))
	cmd.AddCommand(cmdFieldCreate.NewCmdCreateField(f, nil))
	cmd.AddCommand(cmdFieldDelete.NewCmdDeleteField(f, nil))

	return cmd
}
