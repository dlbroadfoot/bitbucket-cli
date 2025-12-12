package create

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

func TestNewCmdCreate(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    CreateOptions
		wantsErr string
	}{
		{
			name: "title only",
			cli:  "--title 'My Issue'",
			wants: CreateOptions{
				Title:    "My Issue",
				Kind:     "bug",
				Priority: "major",
			},
		},
		{
			name: "title and body",
			cli:  "--title 'My Issue' --body 'Description here'",
			wants: CreateOptions{
				Title:    "My Issue",
				Body:     "Description here",
				Kind:     "bug",
				Priority: "major",
			},
		},
		{
			name: "with kind",
			cli:  "--title 'Feature' --kind enhancement",
			wants: CreateOptions{
				Title:    "Feature",
				Kind:     "enhancement",
				Priority: "major",
			},
		},
		{
			name: "with priority",
			cli:  "--title 'Urgent' --priority critical",
			wants: CreateOptions{
				Title:    "Urgent",
				Kind:     "bug",
				Priority: "critical",
			},
		},
		{
			name: "with assignee",
			cli:  "--title 'Task' --assignee johndoe",
			wants: CreateOptions{
				Title:    "Task",
				Kind:     "bug",
				Priority: "major",
				Assignee: "johndoe",
			},
		},
		{
			name: "all flags",
			cli:  "--title 'Full' --body 'desc' --kind task --priority minor --assignee dev",
			wants: CreateOptions{
				Title:    "Full",
				Body:     "desc",
				Kind:     "task",
				Priority: "minor",
				Assignee: "dev",
			},
		},
		{
			name: "short flags",
			cli:  "-t 'My Issue' -b 'desc' -k proposal -p trivial -a alice",
			wants: CreateOptions{
				Title:    "My Issue",
				Body:     "desc",
				Kind:     "proposal",
				Priority: "trivial",
				Assignee: "alice",
			},
		},
		{
			name:     "missing title",
			cli:      "--body 'desc'",
			wantsErr: "required flag(s) \"title\" not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *CreateOptions
			cmd := NewCmdCreate(f, func(opts *CreateOptions) error {
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

			assert.Equal(t, tt.wants.Title, gotOpts.Title)
			assert.Equal(t, tt.wants.Body, gotOpts.Body)
			assert.Equal(t, tt.wants.Kind, gotOpts.Kind)
			assert.Equal(t, tt.wants.Priority, gotOpts.Priority)
			assert.Equal(t, tt.wants.Assignee, gotOpts.Assignee)
		})
	}
}
