package view

import (
	"bytes"
	"testing"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdView(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ViewOptions
		wantsErr bool
	}{
		{
			name: "no args",
			cli:  "",
			wants: ViewOptions{
				RepoArg: "",
				Web:     false,
			},
		},
		{
			name: "sets repo arg",
			cli:  "some/repo",
			wants: ViewOptions{
				RepoArg: "some/repo",
				Web:     false,
			},
		},
		{
			name: "sets web",
			cli:  "-w",
			wants: ViewOptions{
				RepoArg: "",
				Web:     true,
			},
		},
		{
			name: "sets branch",
			cli:  "-b feat/awesome",
			wants: ViewOptions{
				RepoArg: "",
				Branch:  "feat/awesome",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, _, _ := iostreams.Test()

			f := &cmdutil.Factory{
				IOStreams: io,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ViewOptions
			cmd := NewCmdView(f, func(opts *ViewOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.Web, gotOpts.Web)
			assert.Equal(t, tt.wants.Branch, gotOpts.Branch)
			assert.Equal(t, tt.wants.RepoArg, gotOpts.RepoArg)
		})
	}
}
