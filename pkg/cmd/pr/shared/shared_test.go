package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePRArg(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantNum    int
		wantRepo   bool
		wantHost   string
		wantWS     string
		wantSlug   string
		wantErrMsg string
	}{
		{
			name:    "number only",
			arg:     "123",
			wantNum: 123,
		},
		{
			name:    "zero",
			arg:     "0",
			wantNum: 0,
		},
		{
			name:     "full URL",
			arg:      "https://bitbucket.org/myworkspace/myrepo/pull-requests/456",
			wantNum:  456,
			wantRepo: true,
			wantHost: "bitbucket.org",
			wantWS:   "myworkspace",
			wantSlug: "myrepo",
		},
		{
			name:     "URL with different host",
			arg:      "https://bb.example.com/team/project/pull-requests/789",
			wantNum:  789,
			wantRepo: true,
			wantHost: "bb.example.com",
			wantWS:   "team",
			wantSlug: "project",
		},
		{
			name:       "invalid string",
			arg:        "not-a-number",
			wantErrMsg: "invalid pull request argument: not-a-number",
		},
		{
			name:       "invalid URL",
			arg:        "https://bitbucket.org/myworkspace/myrepo/src/main",
			wantErrMsg: "invalid pull request argument: https://bitbucket.org/myworkspace/myrepo/src/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, repo, err := ParsePRArg(tt.arg)

			if tt.wantErrMsg != "" {
				assert.EqualError(t, err, tt.wantErrMsg)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantNum, num)

			if tt.wantRepo {
				assert.NotNil(t, repo)
				assert.Equal(t, tt.wantHost, repo.RepoHost())
				assert.Equal(t, tt.wantWS, repo.RepoWorkspace())
				assert.Equal(t, tt.wantSlug, repo.RepoSlug())
			} else {
				assert.Nil(t, repo)
			}
		})
	}
}

func TestPRStateFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"open", "OPEN"},
		{"OPEN", "OPEN"},
		{"merged", "MERGED"},
		{"declined", "DECLINED"},
		{"closed", "DECLINED"},
		{"superseded", "SUPERSEDED"},
		{"all", ""},
		{"unknown", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := PRStateFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPullRequest_Methods(t *testing.T) {
	pr := &PullRequest{
		ID:          123,
		Title:       "Test PR",
		Description: "Test description",
		State:       "OPEN",
		Source: Branch{
			Branch: struct {
				Name string `json:"name"`
			}{Name: "feature-branch"},
		},
		Destination: Branch{
			Branch: struct {
				Name string `json:"name"`
			}{Name: "main"},
		},
		Links: Links{
			HTML: struct {
				Href string `json:"href"`
			}{Href: "https://bitbucket.org/workspace/repo/pull-requests/123"},
		},
	}

	assert.Equal(t, "Open", pr.StateDisplay())
	assert.Equal(t, "feature-branch", pr.HeadBranch())
	assert.Equal(t, "main", pr.BaseBranch())
	assert.Equal(t, "https://bitbucket.org/workspace/repo/pull-requests/123", pr.HTMLURL())
}

func TestPullRequest_StateDisplay(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"OPEN", "Open"},
		{"MERGED", "Merged"},
		{"DECLINED", "Declined"},
		{"SUPERSEDED", "Superseded"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			pr := &PullRequest{State: tt.state}
			assert.Equal(t, tt.expected, pr.StateDisplay())
		})
	}
}
