package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
)

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
	Author      User   `json:"author"`
	Source      Branch `json:"source"`
	Destination Branch `json:"destination"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	CloseSource bool   `json:"close_source_branch"`
	Links       Links  `json:"links"`
	Reviewers   []User `json:"reviewers"`
	MergeCommit *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit,omitempty"`
	CommentCount int `json:"comment_count"`
	TaskCount    int `json:"task_count"`
}

type User struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
	Nickname    string `json:"nickname"`
	Links       Links  `json:"links"`
}

type Branch struct {
	Branch struct {
		Name string `json:"name"`
	} `json:"branch"`
	Commit struct {
		Hash string `json:"hash"`
	} `json:"commit"`
	Repository struct {
		FullName string `json:"full_name"`
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		Links    Links  `json:"links"`
	} `json:"repository"`
}

type Links struct {
	HTML struct {
		Href string `json:"href"`
	} `json:"html"`
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
	Avatar struct {
		Href string `json:"href"`
	} `json:"avatar"`
}

// PullRequestList represents a paginated list of pull requests
type PullRequestList struct {
	Size     int           `json:"size"`
	Page     int           `json:"page"`
	PageLen  int           `json:"pagelen"`
	Next     string        `json:"next"`
	Previous string        `json:"previous"`
	Values   []PullRequest `json:"values"`
}

// StateDisplay returns a human-readable state
func (pr *PullRequest) StateDisplay() string {
	switch pr.State {
	case "OPEN":
		return "Open"
	case "MERGED":
		return "Merged"
	case "DECLINED":
		return "Declined"
	case "SUPERSEDED":
		return "Superseded"
	default:
		return pr.State
	}
}

// HeadBranch returns the source branch name
func (pr *PullRequest) HeadBranch() string {
	return pr.Source.Branch.Name
}

// BaseBranch returns the destination branch name
func (pr *PullRequest) BaseBranch() string {
	return pr.Destination.Branch.Name
}

// HTMLURL returns the web URL for the PR
func (pr *PullRequest) HTMLURL() string {
	return pr.Links.HTML.Href
}

// ParsePRArg parses a PR argument which can be a number or URL
func ParsePRArg(arg string) (int, bbrepo.Interface, error) {
	// Try parsing as a number first
	if num, err := strconv.Atoi(arg); err == nil {
		return num, nil, nil
	}

	// Try parsing as a URL
	// Format: https://bitbucket.org/WORKSPACE/REPO/pull-requests/123
	re := regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)/pull-requests/(\d+)`)
	matches := re.FindStringSubmatch(arg)
	if matches != nil {
		host := matches[1]
		workspace := matches[2]
		repoSlug := matches[3]
		num, _ := strconv.Atoi(matches[4])

		repo := bbrepo.NewWithHost(workspace, repoSlug, host)
		return num, repo, nil
	}

	return 0, nil, fmt.Errorf("invalid pull request argument: %s", arg)
}

// PRStateFromString converts a string state to the Bitbucket API state format
func PRStateFromString(state string) string {
	switch strings.ToLower(state) {
	case "open":
		return "OPEN"
	case "merged":
		return "MERGED"
	case "declined", "closed":
		return "DECLINED"
	case "superseded":
		return "SUPERSEDED"
	case "all":
		return ""
	default:
		return strings.ToUpper(state)
	}
}

// Comment represents a Bitbucket pull request comment
type Comment struct {
	ID        int    `json:"id"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Content   struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
	} `json:"content"`
	User   User `json:"user"`
	Inline *struct {
		Path string `json:"path"`
		From *int   `json:"from"`
		To   *int   `json:"to"`
	} `json:"inline,omitempty"`
	Parent *struct {
		ID int `json:"id"`
	} `json:"parent,omitempty"`
	Deleted bool   `json:"deleted"`
	Links   Links  `json:"links"`
}

// CommentList represents a paginated list of comments
type CommentList struct {
	Size     int       `json:"size"`
	Page     int       `json:"page"`
	PageLen  int       `json:"pagelen"`
	Next     string    `json:"next"`
	Previous string    `json:"previous"`
	Values   []Comment `json:"values"`
}
