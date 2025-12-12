package api

import "time"

// Bitbucket API v2.0 Types
// See: https://developer.atlassian.com/cloud/bitbucket/rest/intro/

// PaginatedResponse represents a paginated response from the Bitbucket API.
// All list endpoints return this structure with a "values" array and pagination info.
type PaginatedResponse[T any] struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Values   []T    `json:"values"`
}

// User represents a Bitbucket user account.
type User struct {
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	Type        string `json:"type"` // "user"
	Links       Links  `json:"links"`
}

// Workspace represents a Bitbucket workspace (formerly team).
type Workspace struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Type    string `json:"type"` // "workspace"
	Links   Links  `json:"links"`
	IsAdmin bool   `json:"is_admin,omitempty"`
}

// Project represents a Bitbucket project within a workspace.
type Project struct {
	UUID        string    `json:"uuid"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPrivate   bool      `json:"is_private"`
	Type        string    `json:"type"` // "project"
	Owner       Workspace `json:"owner"`
	Links       Links     `json:"links"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
}

// Repository represents a Bitbucket repository.
type Repository struct {
	UUID        string     `json:"uuid"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	FullName    string     `json:"full_name"` // workspace/repo_slug
	Description string     `json:"description"`
	IsPrivate   bool       `json:"is_private"`
	ForkPolicy  string     `json:"fork_policy"` // "allow_forks", "no_public_forks", "no_forks"
	Language    string     `json:"language"`
	Type        string     `json:"type"` // "repository"
	Project     *Project   `json:"project,omitempty"`
	Workspace   Workspace  `json:"workspace"`
	Owner       User       `json:"owner"`
	MainBranch  *Branch    `json:"mainbranch,omitempty"`
	Parent      *Repository `json:"parent,omitempty"` // For forks
	Links       Links      `json:"links"`
	Size        int64      `json:"size"`
	CreatedOn   time.Time  `json:"created_on"`
	UpdatedOn   time.Time  `json:"updated_on"`
	SCM         string     `json:"scm"` // "git" or "hg"
	HasIssues   bool       `json:"has_issues"`
	HasWiki     bool       `json:"has_wiki"`
}

// Branch represents a git branch.
type Branch struct {
	Name   string  `json:"name"`
	Type   string  `json:"type"` // "branch", "named_branch"
	Target *Commit `json:"target,omitempty"`
	Links  Links   `json:"links"`
}

// Commit represents a git commit.
type Commit struct {
	Hash       string    `json:"hash"`
	Type       string    `json:"type"` // "commit"
	Message    string    `json:"message"`
	Author     Author    `json:"author"`
	Date       time.Time `json:"date"`
	Parents    []Commit  `json:"parents,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
	Links      Links     `json:"links"`
}

// Author represents a commit author (may or may not be linked to a user).
type Author struct {
	Raw  string `json:"raw"` // "Name <email>"
	User *User  `json:"user,omitempty"`
}

// PullRequest represents a Bitbucket pull request.
type PullRequest struct {
	ID                int             `json:"id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	State             PullRequestState `json:"state"`
	Type              string          `json:"type"` // "pullrequest"
	Author            User            `json:"author"`
	Source            PullRequestRef  `json:"source"`
	Destination       PullRequestRef  `json:"destination"`
	MergeCommit       *Commit         `json:"merge_commit,omitempty"`
	CloseSourceBranch bool            `json:"close_source_branch"`
	ClosedBy          *User           `json:"closed_by,omitempty"`
	Reason            string          `json:"reason,omitempty"` // Decline reason
	CreatedOn         time.Time       `json:"created_on"`
	UpdatedOn         time.Time       `json:"updated_on"`
	Links             Links           `json:"links"`
	CommentCount      int             `json:"comment_count"`
	TaskCount         int             `json:"task_count"`
	Reviewers         []User          `json:"reviewers,omitempty"`
	Participants      []Participant   `json:"participants,omitempty"`
}

// PullRequestState represents the state of a pull request.
type PullRequestState string

const (
	PullRequestStateOpen       PullRequestState = "OPEN"
	PullRequestStateMerged     PullRequestState = "MERGED"
	PullRequestStateDeclined   PullRequestState = "DECLINED"
	PullRequestStateSuperseded PullRequestState = "SUPERSEDED"
)

// PullRequestRef represents a branch reference in a pull request.
type PullRequestRef struct {
	Branch     Branch      `json:"branch"`
	Commit     Commit      `json:"commit"`
	Repository *Repository `json:"repository,omitempty"`
}

// Participant represents a participant in a pull request.
type Participant struct {
	User               User   `json:"user"`
	Role               string `json:"role"`        // "PARTICIPANT", "REVIEWER"
	Approved           bool   `json:"approved"`
	State              string `json:"state"`       // "approved", "changes_requested", null
	ParticipatedOn     time.Time `json:"participated_on"`
}

// PullRequestComment represents a comment on a pull request.
type PullRequestComment struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"` // "pullrequest_comment"
	Content   Content   `json:"content"`
	User      User      `json:"user"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Parent    *PullRequestComment `json:"parent,omitempty"`
	Inline    *InlineComment      `json:"inline,omitempty"`
	Links     Links     `json:"links"`
	Deleted   bool      `json:"deleted"`
	Pending   bool      `json:"pending"`
}

// InlineComment represents an inline comment position.
type InlineComment struct {
	From int    `json:"from,omitempty"`
	To   int    `json:"to,omitempty"`
	Path string `json:"path"`
}

// Content represents the content of a comment.
type Content struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"` // "markdown", "creole", "plaintext"
	HTML   string `json:"html"`
}

// Issue represents a Bitbucket issue.
type Issue struct {
	ID         int       `json:"id"`
	Title      string    `json:"title"`
	Content    Content   `json:"content"`
	State      string    `json:"state"`    // "new", "open", "resolved", "on hold", "invalid", "duplicate", "wontfix", "closed"
	Priority   string    `json:"priority"` // "trivial", "minor", "major", "critical", "blocker"
	Kind       string    `json:"kind"`     // "bug", "enhancement", "proposal", "task"
	Type       string    `json:"type"`     // "issue"
	Reporter   User      `json:"reporter"`
	Assignee   *User     `json:"assignee,omitempty"`
	Component  *Component `json:"component,omitempty"`
	Milestone  *Milestone `json:"milestone,omitempty"`
	Version    *Version   `json:"version,omitempty"`
	Votes      int       `json:"votes"`
	Watches    int       `json:"watches"`
	Repository Repository `json:"repository"`
	Links      Links     `json:"links"`
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
	EditedOn   *time.Time `json:"edited_on,omitempty"`
}

// Component represents an issue component.
type Component struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Links Links `json:"links"`
}

// Milestone represents an issue milestone.
type Milestone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Links Links `json:"links"`
}

// Version represents an issue version.
type Version struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Links Links `json:"links"`
}

// IssueComment represents a comment on an issue.
type IssueComment struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"` // "issue_comment"
	Content   Content   `json:"content"`
	User      User      `json:"user"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Links     Links     `json:"links"`
}

// Links represents hypermedia links in API responses.
type Links struct {
	Self        *Link `json:"self,omitempty"`
	HTML        *Link `json:"html,omitempty"`
	Avatar      *Link `json:"avatar,omitempty"`
	Clone       []CloneLink `json:"clone,omitempty"`
	Commits     *Link `json:"commits,omitempty"`
	Watchers    *Link `json:"watchers,omitempty"`
	Branches    *Link `json:"branches,omitempty"`
	Tags        *Link `json:"tags,omitempty"`
	Forks       *Link `json:"forks,omitempty"`
	Downloads   *Link `json:"downloads,omitempty"`
	PullRequests *Link `json:"pullrequests,omitempty"`
	Issues      *Link `json:"issues,omitempty"`
	Diff        *Link `json:"diff,omitempty"`
	DiffStat    *Link `json:"diffstat,omitempty"`
	Patch       *Link `json:"patch,omitempty"`
	Comments    *Link `json:"comments,omitempty"`
	Approve     *Link `json:"approve,omitempty"`
	Merge       *Link `json:"merge,omitempty"`
	Decline     *Link `json:"decline,omitempty"`
	Activity    *Link `json:"activity,omitempty"`
	Statuses    *Link `json:"statuses,omitempty"`
}

// Link represents a single hypermedia link.
type Link struct {
	Href string `json:"href"`
	Name string `json:"name,omitempty"`
}

// CloneLink represents a clone URL.
type CloneLink struct {
	Href string `json:"href"`
	Name string `json:"name"` // "https", "ssh"
}

// ErrorResponse represents an error response from the Bitbucket API.
type ErrorResponse struct {
	Type  string `json:"type"` // "error"
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail,omitempty"`
		Data    map[string]interface{} `json:"data,omitempty"`
	} `json:"error"`
}

// CreatePullRequestInput represents the input for creating a pull request.
type CreatePullRequestInput struct {
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	Source            CreatePullRequestRef `json:"source"`
	Destination       *CreatePullRequestRef `json:"destination,omitempty"` // Optional, defaults to main branch
	CloseSourceBranch bool   `json:"close_source_branch,omitempty"`
	Reviewers         []UserRef `json:"reviewers,omitempty"`
}

// CreatePullRequestRef represents a branch reference for PR creation.
type CreatePullRequestRef struct {
	Branch BranchRef `json:"branch"`
}

// BranchRef represents a branch by name.
type BranchRef struct {
	Name string `json:"name"`
}

// UserRef represents a user reference by UUID or account_id.
type UserRef struct {
	UUID      string `json:"uuid,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}

// CreateIssueInput represents the input for creating an issue.
type CreateIssueInput struct {
	Title    string `json:"title"`
	Content  *ContentInput `json:"content,omitempty"`
	Kind     string `json:"kind,omitempty"`     // "bug", "enhancement", "proposal", "task"
	Priority string `json:"priority,omitempty"` // "trivial", "minor", "major", "critical", "blocker"
	Assignee *UserRef `json:"assignee,omitempty"`
}

// ContentInput represents content input for issues/comments.
type ContentInput struct {
	Raw string `json:"raw"`
}

// CreateProjectInput represents the input for creating a project.
type CreateProjectInput struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private,omitempty"`
}

// MergePullRequestInput represents the input for merging a pull request.
type MergePullRequestInput struct {
	CloseSourceBranch bool   `json:"close_source_branch,omitempty"`
	MergeStrategy     string `json:"merge_strategy,omitempty"` // "merge_commit", "squash", "fast_forward"
	Message           string `json:"message,omitempty"`
}

// DeclinePullRequestInput represents the input for declining a pull request.
type DeclinePullRequestInput struct {
	Reason string `json:"reason,omitempty"`
}
