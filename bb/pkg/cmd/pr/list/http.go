package list

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/pkg/cmd/pr/shared"
)

func fetchPullRequests(client *http.Client, repo bbrepo.Interface, opts *ListOptions) ([]shared.PullRequest, error) {
	apiClient := api.NewClientFromHTTP(client)

	// Build query parameters
	params := url.Values{}
	params.Set("pagelen", fmt.Sprintf("%d", opts.Limit))

	// Build query string for filtering
	var queryParts []string

	// Filter by state
	state := shared.PRStateFromString(opts.State)
	if state != "" {
		queryParts = append(queryParts, fmt.Sprintf(`state="%s"`, state))
	}

	// Filter by author
	if opts.Author != "" {
		queryParts = append(queryParts, fmt.Sprintf(`author.nickname="%s"`, opts.Author))
	}

	if len(queryParts) > 0 {
		q := ""
		for i, part := range queryParts {
			if i > 0 {
				q += " AND "
			}
			q += part
		}
		params.Set("q", q)
	}

	path := fmt.Sprintf("repositories/%s/%s/pullrequests?%s",
		repo.RepoWorkspace(), repo.RepoSlug(), params.Encode())

	var result shared.PullRequestList
	err := apiClient.Get(repo.RepoHost(), path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

// FetchPullRequest fetches a single pull request by ID
func FetchPullRequest(client *http.Client, repo bbrepo.Interface, prID int) (*shared.PullRequest, error) {
	path := fmt.Sprintf("repositories/%s/%s/pullrequests/%d",
		repo.RepoWorkspace(), repo.RepoSlug(), prID)

	apiURL := api.RESTPrefix(repo.RepoHost()) + path

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("pull request #%d not found", prID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, api.HandleHTTPError(resp)
	}

	var pr shared.PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

// FetchPullRequestComments fetches comments for a pull request
func FetchPullRequestComments(client *http.Client, repo bbrepo.Interface, prID int) ([]shared.Comment, error) {
	path := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments?pagelen=100",
		repo.RepoWorkspace(), repo.RepoSlug(), prID)

	apiURL := api.RESTPrefix(repo.RepoHost()) + path

	var allComments []shared.Comment

	for apiURL != "" {
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

		if resp.StatusCode != http.StatusOK {
			return nil, api.HandleHTTPError(resp)
		}

		var result shared.CommentList
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		allComments = append(allComments, result.Values...)
		apiURL = result.Next
	}

	return allComments, nil
}
