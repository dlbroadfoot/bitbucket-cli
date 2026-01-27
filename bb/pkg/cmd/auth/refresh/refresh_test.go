package refresh

import (
	"bytes"
	"testing"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/require"
)

func Test_NewCmdRefresh(t *testing.T) {
	tests := []struct {
		name        string
		cli         string
		wants       RefreshOptions
		wantsErr    bool
		tty         bool
		neverPrompt bool
	}{
		{
			name: "tty no arguments",
			tty:  true,
			wants: RefreshOptions{
				Hostname: "",
			},
		},
		{
			name:     "nontty no arguments",
			wantsErr: true,
		},
		{
			name: "nontty hostname",
			cli:  "-h aline.cedrac",
			wants: RefreshOptions{
				Hostname: "aline.cedrac",
			},
		},
		{
			name: "tty hostname",
			tty:  true,
			cli:  "-h aline.cedrac",
			wants: RefreshOptions{
				Hostname: "aline.cedrac",
			},
		},
		{
			name:        "prompts disabled, no args",
			tty:         true,
			cli:         "",
			neverPrompt: true,
			wantsErr:    true,
		},
		{
			name:        "prompts disabled, hostname",
			tty:         true,
			cli:         "-h aline.cedrac",
			neverPrompt: true,
			wants: RefreshOptions{
				Hostname: "aline.cedrac",
			},
		},
		{
			name:  "insecure storage",
			tty:   true,
			cli:   "--insecure-storage",
			wants: RefreshOptions{InsecureStorage: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}
			ios.SetStdinTTY(tt.tty)
			ios.SetStdoutTTY(tt.tty)
			ios.SetNeverPrompt(tt.neverPrompt)

			argv, err := shlex.Split(tt.cli)
			require.NoError(t, err)

			var gotOpts *RefreshOptions
			cmd := NewCmdRefresh(f, func(opts *RefreshOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.Flags().BoolP("help", "x", false, "")

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wants.Hostname, gotOpts.Hostname)
			require.Equal(t, tt.wants.InsecureStorage, gotOpts.InsecureStorage)
		})
	}
}
