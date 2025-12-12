package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIssueArg(t *testing.T) {
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
			arg:      "https://bitbucket.org/myworkspace/myrepo/issues/456",
			wantNum:  456,
			wantRepo: true,
			wantHost: "bitbucket.org",
			wantWS:   "myworkspace",
			wantSlug: "myrepo",
		},
		{
			name:     "URL with different host",
			arg:      "https://bb.example.com/team/project/issues/789",
			wantNum:  789,
			wantRepo: true,
			wantHost: "bb.example.com",
			wantWS:   "team",
			wantSlug: "project",
		},
		{
			name:       "invalid string",
			arg:        "not-a-number",
			wantErrMsg: "invalid issue argument: not-a-number",
		},
		{
			name:       "invalid URL - wrong path",
			arg:        "https://bitbucket.org/myworkspace/myrepo/src/main",
			wantErrMsg: "invalid issue argument: https://bitbucket.org/myworkspace/myrepo/src/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, repo, err := ParseIssueArg(tt.arg)

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

func TestIssueStateFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"new", "new"},
		{"open", "open"},
		{"resolved", "resolved"},
		{"on hold", "on hold"},
		{"onhold", "on hold"},
		{"invalid", "invalid"},
		{"duplicate", "duplicate"},
		{"wontfix", "wontfix"},
		{"won't fix", "wontfix"},
		{"closed", "closed"},
		{"all", ""},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IssueStateFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIssue_Methods(t *testing.T) {
	issue := &Issue{
		ID:       123,
		Title:    "Test Issue",
		State:    "open",
		Priority: "major",
		Kind:     "bug",
		Content: &Content{
			Raw: "Issue description",
		},
		Links: Links{
			HTML: struct {
				Href string `json:"href"`
			}{Href: "https://bitbucket.org/workspace/repo/issues/123"},
		},
	}

	assert.Equal(t, "Open", issue.StateDisplay())
	assert.Equal(t, "Major", issue.PriorityDisplay())
	assert.Equal(t, "Bug", issue.KindDisplay())
	assert.Equal(t, "Issue description", issue.Body())
	assert.Equal(t, "https://bitbucket.org/workspace/repo/issues/123", issue.HTMLURL())
}

func TestIssue_StateDisplay(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"new", "New"},
		{"open", "Open"},
		{"resolved", "Resolved"},
		{"on hold", "On Hold"},
		{"invalid", "Invalid"},
		{"duplicate", "Duplicate"},
		{"wontfix", "Won't Fix"},
		{"closed", "Closed"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			issue := &Issue{State: tt.state}
			assert.Equal(t, tt.expected, issue.StateDisplay())
		})
	}
}

func TestIssue_PriorityDisplay(t *testing.T) {
	tests := []struct {
		priority string
		expected string
	}{
		{"trivial", "Trivial"},
		{"minor", "Minor"},
		{"major", "Major"},
		{"critical", "Critical"},
		{"blocker", "Blocker"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			issue := &Issue{Priority: tt.priority}
			assert.Equal(t, tt.expected, issue.PriorityDisplay())
		})
	}
}

func TestIssue_KindDisplay(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{"bug", "Bug"},
		{"enhancement", "Enhancement"},
		{"proposal", "Proposal"},
		{"task", "Task"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			issue := &Issue{Kind: tt.kind}
			assert.Equal(t, tt.expected, issue.KindDisplay())
		})
	}
}

func TestIssue_BodyNilContent(t *testing.T) {
	issue := &Issue{
		ID:      123,
		Title:   "Test Issue",
		Content: nil,
	}

	assert.Equal(t, "", issue.Body())
}
