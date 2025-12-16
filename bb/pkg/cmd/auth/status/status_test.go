package status

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
)

func Test_NewCmdStatus(t *testing.T) {
	tests := []struct {
		name  string
		cli   string
		wants StatusOptions
	}{
		{
			name:  "no arguments",
			cli:   "",
			wants: StatusOptions{},
		},
		{
			name: "hostname set",
			cli:  "--hostname bitbucket.example.com",
			wants: StatusOptions{
				Hostname: "bitbucket.example.com",
			},
		},
		{
			name: "show token",
			cli:  "--show-token",
			wants: StatusOptions{
				ShowToken: true,
			},
		},
		{
			name: "active",
			cli:  "--active",
			wants: StatusOptions{
				Active: true,
			},
		},
		{
			name: "short flags",
			cli:  "-h bb.io -t -a",
			wants: StatusOptions{
				Hostname:  "bb.io",
				ShowToken: true,
				Active:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &cmdutil.Factory{}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *StatusOptions
			cmd := NewCmdStatus(f, func(opts *StatusOptions) error {
				gotOpts = opts
				return nil
			})

			// TODO cobra hack-around
			cmd.Flags().BoolP("help", "x", false, "")

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.Hostname, gotOpts.Hostname)
			assert.Equal(t, tt.wants.ShowToken, gotOpts.ShowToken)
			assert.Equal(t, tt.wants.Active, gotOpts.Active)
		})
	}
}

func Test_maskToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "email:token format",
			token: "user@example.com:ATATT3xFfGF0m_Z9mwCeNd7q",
			want:  "user@example.com:ATAT********************",
		},
		{
			name:  "short token after colon",
			token: "user@example.com:abc",
			want:  "user@example.com:***",
		},
		{
			name:  "no colon",
			token: "simpletoken12345",
			want:  "simp************",
		},
		{
			name:  "short token no colon",
			token: "abc",
			want:  "***",
		},
		{
			name:  "empty token",
			token: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskToken(tt.token)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_authTokenWriteable(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name:   "BB_TOKEN env var",
			source: "BB_TOKEN",
			want:   false,
		},
		{
			name:   "BITBUCKET_TOKEN env var",
			source: "BITBUCKET_TOKEN",
			want:   false,
		},
		{
			name:   "keyring source",
			source: "keyring",
			want:   true,
		},
		{
			name:   "config file",
			source: "/home/user/.config/bb/hosts.yml",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authTokenWriteable(tt.source)
			require.Equal(t, tt.want, got)
		})
	}
}
