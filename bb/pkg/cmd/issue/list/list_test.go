package list

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
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
			name: "with state resolved",
			cli:  "--state resolved",
			wants: ListOptions{
				State: "resolved",
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
			name: "with kind bug",
			cli:  "--kind bug",
			wants: ListOptions{
				State: "open",
				Kind:  "bug",
				Limit: 30,
			},
		},
		{
			name: "with kind enhancement",
			cli:  "--kind enhancement",
			wants: ListOptions{
				State: "open",
				Kind:  "enhancement",
				Limit: 30,
			},
		},
		{
			name: "with priority critical",
			cli:  "--priority critical",
			wants: ListOptions{
				State:    "open",
				Priority: "critical",
				Limit:    30,
			},
		},
		{
			name: "with assignee",
			cli:  "--assignee johndoe",
			wants: ListOptions{
				State:    "open",
				Assignee: "johndoe",
				Limit:    30,
			},
		},
		{
			name: "with reporter",
			cli:  "--reporter janedoe",
			wants: ListOptions{
				State:    "open",
				Reporter: "janedoe",
				Limit:    30,
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
			name: "short flags",
			cli:  "-s resolved -k bug -p major -a dev",
			wants: ListOptions{
				State:    "resolved",
				Kind:     "bug",
				Priority: "major",
				Assignee: "dev",
				Limit:    30,
			},
		},
		{
			name: "all flags combined",
			cli:  "--state closed --kind task --priority blocker --assignee alice --reporter bob -L 100",
			wants: ListOptions{
				State:    "closed",
				Kind:     "task",
				Priority: "blocker",
				Assignee: "alice",
				Reporter: "bob",
				Limit:    100,
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
			assert.Equal(t, tt.wants.Kind, gotOpts.Kind)
			assert.Equal(t, tt.wants.Priority, gotOpts.Priority)
			assert.Equal(t, tt.wants.Assignee, gotOpts.Assignee)
			assert.Equal(t, tt.wants.Reporter, gotOpts.Reporter)
			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
		})
	}
}
