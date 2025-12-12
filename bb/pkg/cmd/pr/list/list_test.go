package list

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cli/bb/v2/pkg/cmdutil"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ListOptions
		wantsErr string
	}{
		{
			name: "no arguments",
			cli:  "",
			wants: ListOptions{
				State: "open",
				Limit: 30,
			},
		},
		{
			name: "with state open",
			cli:  "--state open",
			wants: ListOptions{
				State: "open",
				Limit: 30,
			},
		},
		{
			name: "with state merged",
			cli:  "--state merged",
			wants: ListOptions{
				State: "merged",
				Limit: 30,
			},
		},
		{
			name: "with state all",
			cli:  "--state all",
			wants: ListOptions{
				State: "all",
				Limit: 30,
			},
		},
		{
			name: "with author",
			cli:  "--author johndoe",
			wants: ListOptions{
				State:  "open",
				Author: "johndoe",
				Limit:  30,
			},
		},
		{
			name: "with limit",
			cli:  "-L 50",
			wants: ListOptions{
				State: "open",
				Limit: 50,
			},
		},
		{
			name: "with short state flag",
			cli:  "-s merged",
			wants: ListOptions{
				State: "merged",
				Limit: 30,
			},
		},
		{
			name: "all flags combined",
			cli:  "--state merged --author jane -L 100",
			wants: ListOptions{
				State:  "merged",
				Author: "jane",
				Limit:  100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &cmdutil.Factory{}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr != "" {
				assert.EqualError(t, err, tt.wantsErr)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wants.State, gotOpts.State)
			assert.Equal(t, tt.wants.Author, gotOpts.Author)
			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
		})
	}
}
