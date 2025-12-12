package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cli/bb/v2/internal/bbrepo"
)

// Issue represents a Bitbucket issue
type Issue struct {
	ID        int      `json:"id"`
	Title     string   `json:"title"`
	Content   *Content `json:"content,omitempty"`
	State     string   `json:"state"` // new, open, resolved, on hold, invalid, duplicate, wontfix, closed
	Priority  string   `json:"priority"` // trivial, minor, major, critical, blocker
	Kind      string   `json:"kind"` // bug, enhancement, proposal, task
	Reporter  User     `json:"reporter"`
	Assignee  *User    `json:"assignee,omitempty"`
	CreatedOn string   `json:"created_on"`
	UpdatedOn string   `json:"updated_on"`
	Links     Links    `json:"links"`
	Votes     int      `json:"votes"`
	Watches   int      `json:"watches"`
}

type Content struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

type User struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
	Nickname    string `json:"nickname"`
	Links       Links  `json:"links"`
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

// IssueList represents a paginated list of issues
type IssueList struct {
	Size     int     `json:"size"`
	Page     int     `json:"page"`
	PageLen  int     `json:"pagelen"`
	Next     string  `json:"next"`
	Previous string  `json:"previous"`
	Values   []Issue `json:"values"`
}

// StateDisplay returns a human-readable state
func (i *Issue) StateDisplay() string {
	switch i.State {
	case "new":
		return "New"
	case "open":
		return "Open"
	case "resolved":
		return "Resolved"
	case "on hold":
		return "On Hold"
	case "invalid":
		return "Invalid"
	case "duplicate":
		return "Duplicate"
	case "wontfix":
		return "Won't Fix"
	case "closed":
		return "Closed"
	default:
		return i.State
	}
}

// PriorityDisplay returns a human-readable priority
func (i *Issue) PriorityDisplay() string {
	switch i.Priority {
	case "trivial":
		return "Trivial"
	case "minor":
		return "Minor"
	case "major":
		return "Major"
	case "critical":
		return "Critical"
	case "blocker":
		return "Blocker"
	default:
		return i.Priority
	}
}

// KindDisplay returns a human-readable kind
func (i *Issue) KindDisplay() string {
	switch i.Kind {
	case "bug":
		return "Bug"
	case "enhancement":
		return "Enhancement"
	case "proposal":
		return "Proposal"
	case "task":
		return "Task"
	default:
		return i.Kind
	}
}

// Body returns the issue content
func (i *Issue) Body() string {
	if i.Content != nil {
		return i.Content.Raw
	}
	return ""
}

// HTMLURL returns the web URL for the issue
func (i *Issue) HTMLURL() string {
	return i.Links.HTML.Href
}

// ParseIssueArg parses an issue argument which can be a number or URL
func ParseIssueArg(arg string) (int, bbrepo.Interface, error) {
	// Try parsing as a number first
	if num, err := strconv.Atoi(arg); err == nil {
		return num, nil, nil
	}

	// Try parsing as a URL
	// Format: https://bitbucket.org/WORKSPACE/REPO/issues/123
	re := regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)/issues/(\d+)`)
	matches := re.FindStringSubmatch(arg)
	if matches != nil {
		host := matches[1]
		workspace := matches[2]
		repoSlug := matches[3]
		num, _ := strconv.Atoi(matches[4])

		repo := bbrepo.NewWithHost(workspace, repoSlug, host)
		return num, repo, nil
	}

	return 0, nil, fmt.Errorf("invalid issue argument: %s", arg)
}

// IssueStateFromString converts a string state to the Bitbucket API state format
func IssueStateFromString(state string) string {
	switch strings.ToLower(state) {
	case "new":
		return "new"
	case "open":
		return "open"
	case "resolved":
		return "resolved"
	case "on hold", "onhold":
		return "on hold"
	case "invalid":
		return "invalid"
	case "duplicate":
		return "duplicate"
	case "wontfix", "won't fix":
		return "wontfix"
	case "closed":
		return "closed"
	case "all":
		return ""
	default:
		return strings.ToLower(state)
	}
}
