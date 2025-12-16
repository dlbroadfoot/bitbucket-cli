package login

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"

	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

func Test_NewCmdLogin(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		stdin    string
		stdinTTY bool
		wants    LoginOptions
		wantsErr bool
	}{
		{
			name:  "nontty, with-token",
			stdin: "abc123\n",
			cli:   "--with-token",
			wants: LoginOptions{
				Hostname: "bitbucket.org",
				Token:    "abc123",
			},
		},
		{
			name:     "tty, with-token",
			stdinTTY: true,
			stdin:    "def456",
			cli:      "--with-token",
			wants: LoginOptions{
				Hostname: "bitbucket.org",
				Token:    "def456",
			},
		},
		{
			name:     "nontty, hostname",
			stdinTTY: false,
			cli:      "--hostname bb.example.com",
			wants: LoginOptions{
				Hostname: "bb.example.com",
				Token:    "",
			},
		},
		{
			name:     "nontty, default hostname",
			stdinTTY: false,
			cli:      "",
			wants: LoginOptions{
				Hostname: "bitbucket.org",
				Token:    "",
			},
		},
		{
			name:  "nontty, with-token, hostname",
			cli:   "--hostname bb.example.com --with-token",
			stdin: "abc123\n",
			wants: LoginOptions{
				Hostname: "bb.example.com",
				Token:    "abc123",
			},
		},
		{
			name:     "tty, with-token, hostname",
			stdinTTY: true,
			stdin:    "ghi789",
			cli:      "--with-token --hostname bb.example.com",
			wants: LoginOptions{
				Hostname: "bb.example.com",
				Token:    "ghi789",
			},
		},
		{
			name:     "tty, hostname",
			stdinTTY: true,
			cli:      "--hostname bb.example.com",
			wants: LoginOptions{
				Hostname:    "bb.example.com",
				Token:       "",
				Interactive: true,
			},
		},
		{
			name:     "tty, interactive",
			stdinTTY: true,
			cli:      "",
			wants: LoginOptions{
				Hostname:    "bitbucket.org",
				Token:       "",
				Interactive: true,
			},
		},
		{
			name:     "tty with email",
			stdinTTY: true,
			cli:      "--email user@example.com",
			wants: LoginOptions{
				Hostname:    "bitbucket.org",
				Email:       "user@example.com",
				Interactive: true,
			},
		},
		{
			name:     "nontty with email",
			stdinTTY: false,
			cli:      "--email user@example.com",
			wants: LoginOptions{
				Hostname: "bitbucket.org",
				Email:    "user@example.com",
			},
		},
		{
			name:     "with git protocol https",
			stdinTTY: true,
			cli:      "--git-protocol https",
			wants: LoginOptions{
				Hostname:    "bitbucket.org",
				GitProtocol: "https",
				Interactive: true,
			},
		},
		{
			name:     "with git protocol ssh",
			stdinTTY: true,
			cli:      "--git-protocol ssh",
			wants: LoginOptions{
				Hostname:    "bitbucket.org",
				GitProtocol: "ssh",
				Interactive: true,
			},
		},
		{
			name:     "insecure storage",
			stdinTTY: true,
			cli:      "--insecure-storage",
			wants: LoginOptions{
				Hostname:        "bitbucket.org",
				InsecureStorage: true,
				Interactive:     true,
			},
		},
		{
			name: "nontty insecure storage",
			cli:  "--insecure-storage",
			wants: LoginOptions{
				Hostname:        "bitbucket.org",
				InsecureStorage: true,
			},
		},
		{
			name: "short flags",
			cli:  "-h bb.example.com -e user@example.com -p https",
			wants: LoginOptions{
				Hostname:    "bb.example.com",
				Email:       "user@example.com",
				GitProtocol: "https",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, stdin, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			ios.SetStdoutTTY(true)
			ios.SetStdinTTY(tt.stdinTTY)
			if tt.stdin != "" {
				stdin.WriteString(tt.stdin)
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *LoginOptions
			cmd := NewCmdLogin(f, func(opts *LoginOptions) error {
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
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.Token, gotOpts.Token)
			assert.Equal(t, tt.wants.Hostname, gotOpts.Hostname)
			assert.Equal(t, tt.wants.Email, gotOpts.Email)
			assert.Equal(t, tt.wants.GitProtocol, gotOpts.GitProtocol)
			assert.Equal(t, tt.wants.InsecureStorage, gotOpts.InsecureStorage)
			assert.Equal(t, tt.wants.Interactive, gotOpts.Interactive)
		})
	}
}
