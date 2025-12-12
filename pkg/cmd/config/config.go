package config

import (
	"fmt"
	"strings"

	"github.com/dlbroadfoot/bitbucket-cli/internal/config"
	cmdClearCache "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/config/clear-cache"
	cmdGet "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/config/get"
	cmdList "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/config/list"
	cmdSet "github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/config/set"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdConfig(f *cmdutil.Factory) *cobra.Command {
	longDoc := strings.Builder{}
	longDoc.WriteString("Display or change configuration settings for bb.\n\n")
	longDoc.WriteString("Current respected settings:\n")
	for _, co := range config.Options {
		longDoc.WriteString(fmt.Sprintf("- `%s`: %s", co.Key, co.Description))
		if len(co.AllowedValues) > 0 {
			longDoc.WriteString(fmt.Sprintf(" `{%s}`", strings.Join(co.AllowedValues, " | ")))
		}
		if co.DefaultValue != "" {
			longDoc.WriteString(fmt.Sprintf(" (default `%s`)", co.DefaultValue))
		}
		longDoc.WriteRune('\n')
	}

	cmd := &cobra.Command{
		Use:   "config <command>",
		Short: "Manage configuration for bb",
		Long:  longDoc.String(),
	}

	cmdutil.DisableAuthCheck(cmd)

	cmd.AddCommand(cmdGet.NewCmdConfigGet(f, nil))
	cmd.AddCommand(cmdSet.NewCmdConfigSet(f, nil))
	cmd.AddCommand(cmdList.NewCmdConfigList(f, nil))
	cmd.AddCommand(cmdClearCache.NewCmdConfigClearCache(f, nil))

	return cmd
}
