package search

import (
	"github.com/MakeNowJust/heredoc"
	codeCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/search/code"
	reposCmd "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/search/repos"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <command>",
		Short: "Search for repositories and code",
		Long: heredoc.Doc(`
			Search for repositories and code across Bitbucket.

			You can search within a specific workspace or across all accessible workspaces.
		`),
		Example: heredoc.Doc(`
			$ bb search repos cli --workspace myworkspace
			$ bb search code "func main" --workspace myworkspace
		`),
		GroupID: "core",
	}

	cmd.AddCommand(reposCmd.NewCmdRepos(f, nil))
	cmd.AddCommand(codeCmd.NewCmdCode(f, nil))

	return cmd
}
