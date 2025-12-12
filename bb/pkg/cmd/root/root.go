package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	aliasCmd "github.com/cli/bb/v2/pkg/cmd/alias"
	"github.com/cli/bb/v2/pkg/cmd/alias/shared"
	apiCmd "github.com/cli/bb/v2/pkg/cmd/api"
	authCmd "github.com/cli/bb/v2/pkg/cmd/auth"
	completionCmd "github.com/cli/bb/v2/pkg/cmd/completion"
	configCmd "github.com/cli/bb/v2/pkg/cmd/config"
	"github.com/cli/bb/v2/pkg/cmd/factory"
	issueCmd "github.com/cli/bb/v2/pkg/cmd/issue"
	prCmd "github.com/cli/bb/v2/pkg/cmd/pr"
	projectCmd "github.com/cli/bb/v2/pkg/cmd/project"
	repoCmd "github.com/cli/bb/v2/pkg/cmd/repo"
	versionCmd "github.com/cli/bb/v2/pkg/cmd/version"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

type AuthError struct {
	err error
}

func (ae *AuthError) Error() string {
	return ae.err.Error()
}

func NewCmdRoot(f *cmdutil.Factory, version, buildDate string) (*cobra.Command, error) {
	io := f.IOStreams
	cfg, err := f.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration: %s\n", err)
	}

	cmd := &cobra.Command{
		Use:   "bb <command> <subcommand> [flags]",
		Short: "Bitbucket CLI",
		Long:  `Work seamlessly with Bitbucket from the command line.`,
		Example: heredoc.Doc(`
			$ bb issue create
			$ bb pr list
			$ bb pr checkout 321
		`),
		Annotations: map[string]string{
			"versionInfo": versionCmd.Format(version, buildDate),
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// require that the user is authenticated before running most commands
			if cmdutil.IsAuthCheckEnabled(cmd) && !cmdutil.CheckAuth(cfg) {
				fmt.Fprint(io.ErrOut, authHelp())
				return &AuthError{}
			}
			return nil
		},
	}

	// cmd.SetOut(f.IOStreams.Out)    // can't use due to https://github.com/spf13/cobra/issues/1708
	// cmd.SetErr(f.IOStreams.ErrOut) // just let it default to os.Stderr instead

	cmd.PersistentFlags().Bool("help", false, "Show help for command")

	// override Cobra's default behaviors unless an opt-out has been set
	if os.Getenv("BB_COBRA") == "" {
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		// this --version flag is checked in rootHelpFunc
		cmd.Flags().Bool("version", false, "Show bb version")

		cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
			rootHelpFunc(f, c, args)
		})
		cmd.SetUsageFunc(func(c *cobra.Command) error {
			return rootUsageFunc(f.IOStreams.ErrOut, c)
		})
		cmd.SetFlagErrorFunc(rootFlagErrorFunc)
	}

	cmd.AddGroup(&cobra.Group{
		ID:    "core",
		Title: "Core commands",
	})

	// Child commands
	cmd.AddCommand(versionCmd.NewCmdVersion(f, version, buildDate))
	cmd.AddCommand(aliasCmd.NewCmdAlias(f))
	cmd.AddCommand(authCmd.NewCmdAuth(f))
	cmd.AddCommand(configCmd.NewCmdConfig(f))
	cmd.AddCommand(completionCmd.NewCmdCompletion(f.IOStreams))
	cmd.AddCommand(projectCmd.NewCmdProject(f))

	// below here at the commands that require the "intelligent" BaseRepo resolver
	repoResolvingCmdFactory := *f
	repoResolvingCmdFactory.BaseRepo = factory.SmartBaseRepoFunc(f)

	cmd.AddCommand(prCmd.NewCmdPR(&repoResolvingCmdFactory))
	cmd.AddCommand(issueCmd.NewCmdIssue(&repoResolvingCmdFactory))
	cmd.AddCommand(repoCmd.NewCmdRepo(&repoResolvingCmdFactory))
	cmd.AddCommand(apiCmd.NewCmdApi(&repoResolvingCmdFactory, nil))

	// Help topics
	var referenceCmd *cobra.Command
	for _, ht := range HelpTopics {
		helpTopicCmd := NewCmdHelpTopic(f.IOStreams, ht)
		cmd.AddCommand(helpTopicCmd)

		// See bottom of the function for why we explicitly care about the reference cmd
		if ht.name == "reference" {
			referenceCmd = helpTopicCmd
		}
	}

	// Aliases
	aliases := cfg.Aliases()
	validAliasName := shared.ValidAliasNameFunc(cmd)
	validAliasExpansion := shared.ValidAliasExpansionFunc(cmd)
	for k, v := range aliases.All() {
		aliasName := k
		aliasValue := v
		if validAliasName(aliasName) && validAliasExpansion(aliasValue) {
			split, _ := shlex.Split(aliasName)
			parentCmd, parentArgs, _ := cmd.Find(split)
			if !parentCmd.ContainsGroup("alias") {
				parentCmd.AddGroup(&cobra.Group{
					ID:    "alias",
					Title: "Alias commands",
				})
			}
			if strings.HasPrefix(aliasValue, "!") {
				shellAliasCmd := NewCmdShellAlias(io, parentArgs[0], aliasValue)
				parentCmd.AddCommand(shellAliasCmd)
			} else {
				aliasCmd := NewCmdAlias(io, parentArgs[0], aliasValue)
				split, _ := shlex.Split(aliasValue)
				child, _, _ := cmd.Find(split)
				aliasCmd.SetUsageFunc(func(_ *cobra.Command) error {
					return rootUsageFunc(f.IOStreams.ErrOut, child)
				})
				aliasCmd.SetHelpFunc(func(_ *cobra.Command, args []string) {
					rootHelpFunc(f, child, args)
				})
				parentCmd.AddCommand(aliasCmd)
			}
		}
	}

	cmdutil.DisableAuthCheck(cmd)

	// The reference command produces paged output that displays information on every other command.
	// Therefore, we explicitly set the Long text and HelpFunc here after all other commands are registered.
	// We experimented with producing the paged output dynamically when the HelpFunc is called but since
	// doc generation makes use of the Long text, it is simpler to just be explicit here that this command
	// is special.
	referenceCmd.Long = stringifyReference(cmd)
	referenceCmd.SetHelpFunc(longPager(f.IOStreams))
	return cmd, nil
}
