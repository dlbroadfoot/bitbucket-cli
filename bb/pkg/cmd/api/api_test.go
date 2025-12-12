package api

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
)

func TestNewCmdApi(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ApiOptions
		wantsErr string
	}{
		{
			name: "simple endpoint",
			cli:  "user",
			wants: ApiOptions{
				RequestPath: "user",
			},
		},
		{
			name: "endpoint with path",
			cli:  "repositories/myworkspace/myrepo",
			wants: ApiOptions{
				RequestPath: "repositories/myworkspace/myrepo",
			},
		},
		{
			name: "with method GET",
			cli:  "user -X GET",
			wants: ApiOptions{
				RequestPath:   "user",
				RequestMethod: "GET",
			},
		},
		{
			name: "with method POST",
			cli:  "user -X POST",
			wants: ApiOptions{
				RequestPath:   "user",
				RequestMethod: "POST",
			},
		},
		{
			name: "with raw field",
			cli:  "issues -f title=test",
			wants: ApiOptions{
				RequestPath: "issues",
				RawFields:   []string{"title=test"},
			},
		},
		{
			name: "with magic field",
			cli:  "issues -F count=42",
			wants: ApiOptions{
				RequestPath: "issues",
				MagicFields: []string{"count=42"},
			},
		},
		{
			name: "with header",
			cli:  "user -H 'Accept: text/plain'",
			wants: ApiOptions{
				RequestPath:    "user",
				RequestHeaders: []string{"Accept: text/plain"},
			},
		},
		{
			name: "with hostname",
			cli:  "user --hostname bb.example.com",
			wants: ApiOptions{
				RequestPath: "user",
				Hostname:    "bb.example.com",
			},
		},
		{
			name: "with silent",
			cli:  "user --silent",
			wants: ApiOptions{
				RequestPath: "user",
				Silent:      true,
			},
		},
		{
			name: "with paginate",
			cli:  "repositories --paginate",
			wants: ApiOptions{
				RequestPath: "repositories",
				Paginate:    true,
			},
		},
		{
			name: "with jq",
			cli:  "user -q '.username'",
			wants: ApiOptions{
				RequestPath: "user",
				JQ:          ".username",
			},
		},
		{
			name: "multiple fields",
			cli:  "issues -f title=test -f body=desc -F priority=1",
			wants: ApiOptions{
				RequestPath: "issues",
				RawFields:   []string{"title=test", "body=desc"},
				MagicFields: []string{"priority=1"},
			},
		},
		{
			name: "multiple headers",
			cli:  "user -H 'Accept: text/plain' -H 'X-Custom: value'",
			wants: ApiOptions{
				RequestPath:    "user",
				RequestHeaders: []string{"Accept: text/plain", "X-Custom: value"},
			},
		},
		{
			name:     "no endpoint",
			cli:      "",
			wantsErr: "accepts 1 arg(s), received 0",
		},
		{
			name:     "too many args",
			cli:      "user extra",
			wantsErr: "accepts 1 arg(s), received 2",
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

			var gotOpts *ApiOptions
			cmd := NewCmdApi(f, func(opts *ApiOptions) error {
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

			assert.Equal(t, tt.wants.RequestPath, gotOpts.RequestPath)
			assert.Equal(t, tt.wants.RequestMethod, gotOpts.RequestMethod)
			assert.Equal(t, tt.wants.Hostname, gotOpts.Hostname)
			assert.Equal(t, tt.wants.Silent, gotOpts.Silent)
			assert.Equal(t, tt.wants.Paginate, gotOpts.Paginate)
			assert.Equal(t, tt.wants.JQ, gotOpts.JQ)
			assert.Equal(t, tt.wants.RawFields, gotOpts.RawFields)
			assert.Equal(t, tt.wants.MagicFields, gotOpts.MagicFields)
			assert.Equal(t, tt.wants.RequestHeaders, gotOpts.RequestHeaders)
		})
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHost string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "full URL",
			input:    "https://api.bitbucket.org/2.0/user",
			wantHost: "api.bitbucket.org",
			wantPath: "/2.0/user",
		},
		{
			name:     "URL with query",
			input:    "https://api.bitbucket.org/2.0/repositories?role=member",
			wantHost: "api.bitbucket.org",
			wantPath: "/2.0/repositories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantHost, u.Host)
			assert.Equal(t, tt.wantPath, u.Path)
		})
	}
}
