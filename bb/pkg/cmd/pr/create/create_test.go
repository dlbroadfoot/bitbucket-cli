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
			cli:  "--title 'My PR'",
			wants: CreateOptions{
				Title: "My PR",
			},
		},
		{
			name: "title and body",
			cli:  "--title 'My PR' --body 'Description here'",
			wants: CreateOptions{
				Title: "My PR",
				Body:  "Description here",
			},
		},
		{
			name: "with head branch",
			cli:  "--title 'My PR' --head feature-branch",
			wants: CreateOptions{
				Title:      "My PR",
				HeadBranch: "feature-branch",
			},
		},
		{
			name: "with base branch",
			cli:  "--title 'My PR' --base main",
			wants: CreateOptions{
				Title:      "My PR",
				BaseBranch: "main",
			},
		},
		{
			name: "close source branch",
			cli:  "--title 'My PR' --close-source",
			wants: CreateOptions{
				Title:       "My PR",
				CloseSource: true,
			},
		},
		{
			name: "all flags",
			cli:  "--title 'My PR' --body 'desc' --head feature --base main --close-source",
			wants: CreateOptions{
				Title:       "My PR",
				Body:        "desc",
				HeadBranch:  "feature",
				BaseBranch:  "main",
				CloseSource: true,
			},
		},
		{
			name: "short flags",
			cli:  "-t 'My PR' -b 'desc' -H feature -B main",
			wants: CreateOptions{
				Title:      "My PR",
				Body:       "desc",
				HeadBranch: "feature",
				BaseBranch: "main",
			},
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
			assert.Equal(t, tt.wants.HeadBranch, gotOpts.HeadBranch)
			assert.Equal(t, tt.wants.BaseBranch, gotOpts.BaseBranch)
			assert.Equal(t, tt.wants.CloseSource, gotOpts.CloseSource)
		})
	}
}
