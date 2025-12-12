package list

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/shared"
)

func fetchIssues(client *http.Client, repo bbrepo.Interface, opts *ListOptions) ([]shared.Issue, error) {
	// Build query parameters
	query := url.Values{}

	// Build filter query
	var filters []string

	// State filter
	if opts.State != "" && opts.State != "all" {
		state := shared.IssueStateFromString(opts.State)
		if state != "" {
			filters = append(filters, fmt.Sprintf(`state="%s"`, state))
		}
	}

	// Kind filter
	if opts.Kind != "" {
		filters = append(filters, fmt.Sprintf(`kind="%s"`, strings.ToLower(opts.Kind)))
	}

	// Priority filter
	if opts.Priority != "" {
		filters = append(filters, fmt.Sprintf(`priority="%s"`, strings.ToLower(opts.Priority)))
	}

	// Assignee filter
	if opts.Assignee != "" {
		filters = append(filters, fmt.Sprintf(`assignee.nickname="%s"`, opts.Assignee))
	}

	// Reporter filter
	if opts.Reporter != "" {
		filters = append(filters, fmt.Sprintf(`reporter.nickname="%s"`, opts.Reporter))
	}

	if len(filters) > 0 {
		query.Set("q", strings.Join(filters, " AND "))
	}

	// Pagination
	if opts.Limit > 0 && opts.Limit <= 100 {
		query.Set("pagelen", fmt.Sprintf("%d", opts.Limit))
	}

	// Sort by updated date descending
	query.Set("sort", "-updated_on")

	// Build URL
	apiURL := fmt.Sprintf("%srepositories/%s/%s/issues",
		bbinstance.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
	)
	if len(query) > 0 {
		apiURL += "?" + query.Encode()
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("issue tracker is not enabled for this repository")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result shared.IssueList
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Limit results if needed
	issues := result.Values
	if opts.Limit > 0 && len(issues) > opts.Limit {
		issues = issues[:opts.Limit]
	}

	return issues, nil
}

// FetchIssue fetches a single issue by ID
func FetchIssue(client *http.Client, repo bbrepo.Interface, issueID int) (*shared.Issue, error) {
	apiURL := fmt.Sprintf("%srepositories/%s/%s/issues/%d",
		bbinstance.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		issueID,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("issue #%d not found", issueID)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var issue shared.Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return &issue, nil
}
