package sshkey

import (
	"github.com/MakeNowJust/heredoc"
	addCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/ssh-key/add"
	deleteCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/ssh-key/delete"
	listCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/ssh-key/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdSSHKey(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-key <command>",
		Short: "Manage SSH keys",
		Long: heredoc.Doc(`
			Manage SSH keys for your Bitbucket account.

			SSH keys are used to authenticate with Bitbucket when cloning repositories
			or pushing changes over SSH.
		`),
		Example: heredoc.Doc(`
			$ bb ssh-key list
			$ bb ssh-key add --title "My Laptop"
			$ bb ssh-key delete <key-id>
		`),
		GroupID: "core",
	}

	cmd.AddCommand(listCmd.NewCmdList(f, nil))
	cmd.AddCommand(addCmd.NewCmdAdd(f, nil))
	cmd.AddCommand(deleteCmd.NewCmdDelete(f, nil))

	return cmd
}
