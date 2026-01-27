package clone

import (
	"net/url"
	"testing"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdClone(t *testing.T) {
	testCases := []struct {
		name     string
		args     string
		wantOpts CloneOptions
		wantErr  string
	}{
		{
			name:    "no arguments",
			args:    "",
			wantErr: "cannot clone: repository argument required",
		},
		{
			name: "repo argument",
			args: "OWNER/REPO",
			wantOpts: CloneOptions{
				Repository: "OWNER/REPO",
				GitArgs:    []string{},
			},
		},
		{
			name: "directory argument",
			args: "OWNER/REPO mydir",
			wantOpts: CloneOptions{
				Repository: "OWNER/REPO",
				GitArgs:    []string{"mydir"},
			},
		},
		{
			name: "git clone arguments",
			args: "OWNER/REPO -- --depth 1 --recurse-submodules",
			wantOpts: CloneOptions{
				Repository: "OWNER/REPO",
				GitArgs:    []string{"--depth", "1", "--recurse-submodules"},
			},
		},
		{
			name:    "unknown argument",
			args:    "OWNER/REPO --depth 1",
			wantErr: "unknown flag: --depth\nSeparate git clone flags with '--'.",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ios, stdin, stdout, stderr := iostreams.Test()
			fac := &cmdutil.Factory{IOStreams: ios}

			var opts *CloneOptions
			cmd := NewCmdClone(fac, func(co *CloneOptions) error {
				opts = co
				return nil
			})

			argv, err := shlex.Split(tt.args)
			require.NoError(t, err)
			cmd.SetArgs(argv)

			cmd.SetIn(stdin)
			cmd.SetOut(stderr)
			cmd.SetErr(stderr)

			_, err = cmd.ExecuteC()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, "", stdout.String())
			assert.Equal(t, "", stderr.String())

			assert.Equal(t, tt.wantOpts.Repository, opts.Repository)
			assert.Equal(t, tt.wantOpts.GitArgs, opts.GitArgs)
		})
	}
}

func TestSimplifyURL(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		expectedRaw string
	}{
		{
			name:        "empty",
			raw:         "",
			expectedRaw: "",
		},
		{
			name:        "no change, no path",
			raw:         "https://bitbucket.org",
			expectedRaw: "https://bitbucket.org",
		},
		{
			name:        "no change, single part path",
			raw:         "https://bitbucket.org/owner",
			expectedRaw: "https://bitbucket.org/owner",
		},
		{
			name:        "no change, two-part path",
			raw:         "https://bitbucket.org/owner/repo",
			expectedRaw: "https://bitbucket.org/owner/repo",
		},
		{
			name:        "no change, three-part path",
			raw:         "https://bitbucket.org/owner/repo/pulls",
			expectedRaw: "https://bitbucket.org/owner/repo",
		},
		{
			name:        "no change, two-part path, with query, with fragment",
			raw:         "https://bitbucket.org/owner/repo?key=value#fragment",
			expectedRaw: "https://bitbucket.org/owner/repo",
		},
		{
			name:        "no change, single part path, with query, with fragment",
			raw:         "https://bitbucket.org/owner?key=value#fragment",
			expectedRaw: "https://bitbucket.org/owner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)
			result := simplifyURL(u)
			assert.Equal(t, tt.expectedRaw, result.String())
		})
	}
}
