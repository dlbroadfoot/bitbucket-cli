package list

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/internal/bbinstance"
)

type RepositoryList struct {
	Owner        string
	Repositories []api.Repository
	TotalCount   int
	FromSearch   bool
}

type FilterOptions struct {
	Visibility  string // private, public
	Fork        bool
	Source      bool
	Language    string
	Topic       []string
	Archived    bool
	NonArchived bool
	Fields      []string
}

// listRepos lists repositories for a workspace using Bitbucket REST API.
// Bitbucket API: GET /2.0/repositories/{workspace}
func listRepos(client *http.Client, hostname string, limit int, workspace string, filter FilterOptions) (*RepositoryList, error) {
	if workspace == "" {
		// List repositories for the authenticated user
		return listUserRepos(client, hostname, limit, filter)
	}

	return listWorkspaceRepos(client, hostname, limit, workspace, filter)
}

// listUserRepos lists repositories the authenticated user has access to.
// Bitbucket API: GET /2.0/user/permissions/repositories
func listUserRepos(client *http.Client, hostname string, limit int, filter FilterOptions) (*RepositoryList, error) {
	apiURL := bbinstance.RESTPrefix(hostname) + "user/permissions/repositories"

	params := url.Values{}
	params.Set("pagelen", fmt.Sprintf("%d", min(limit, 100)))

	// Build query filter
	var queryParts []string
	if filter.Visibility == "private" {
		queryParts = append(queryParts, "repository.is_private=true")
	} else if filter.Visibility == "public" {
		queryParts = append(queryParts, "repository.is_private=false")
	}
	if len(queryParts) > 0 {
		params.Set("q", strings.Join(queryParts, " AND "))
	}

	fullURL := apiURL + "?" + params.Encode()

	result := &RepositoryList{}
	for fullURL != "" && len(result.Repositories) < limit {
		req, err := http.NewRequest("GET", fullURL, nil)
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

		var pageResp struct {
			Size   int    `json:"size"`
			Page   int    `json:"page"`
			Next   string `json:"next"`
			Values []struct {
				Repository api.Repository `json:"repository"`
			} `json:"values"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		for _, v := range pageResp.Values {
			repo := v.Repository
			// Apply client-side filters
			if filter.Fork && repo.Parent == nil {
				continue
			}
			if filter.Source && repo.Parent != nil {
				continue
			}
			if filter.Language != "" && !strings.EqualFold(repo.Language, filter.Language) {
				continue
			}

			result.Repositories = append(result.Repositories, repo)
			if result.Owner == "" && repo.Workspace.Slug != "" {
				result.Owner = repo.Workspace.Slug
			}
			if len(result.Repositories) >= limit {
				break
			}
		}

		result.TotalCount = pageResp.Size
		fullURL = pageResp.Next
	}

	return result, nil
}

// listWorkspaceRepos lists repositories in a specific workspace.
// Bitbucket API: GET /2.0/repositories/{workspace}
func listWorkspaceRepos(client *http.Client, hostname string, limit int, workspace string, filter FilterOptions) (*RepositoryList, error) {
	apiURL := bbinstance.RESTPrefix(hostname) + "repositories/" + workspace

	params := url.Values{}
	params.Set("pagelen", fmt.Sprintf("%d", min(limit, 100)))

	// Build query filter
	var queryParts []string
	if filter.Visibility == "private" {
		queryParts = append(queryParts, "is_private=true")
	} else if filter.Visibility == "public" {
		queryParts = append(queryParts, "is_private=false")
	}
	if filter.Language != "" {
		queryParts = append(queryParts, fmt.Sprintf("language=%q", filter.Language))
	}
	if len(queryParts) > 0 {
		params.Set("q", strings.Join(queryParts, " AND "))
	}

	fullURL := apiURL + "?" + params.Encode()

	result := &RepositoryList{Owner: workspace}
	for fullURL != "" && len(result.Repositories) < limit {
		req, err := http.NewRequest("GET", fullURL, nil)
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

		var pageResp api.PaginatedResponse[api.Repository]
		if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		for _, repo := range pageResp.Values {
			// Apply client-side filters
			if filter.Fork && repo.Parent == nil {
				continue
			}
			if filter.Source && repo.Parent != nil {
				continue
			}

			result.Repositories = append(result.Repositories, repo)
			if len(result.Repositories) >= limit {
				break
			}
		}

		result.TotalCount = pageResp.Size
		fullURL = pageResp.Next
	}

	return result, nil
}

// searchRepos searches for repositories (simplified - Bitbucket search is limited)
func searchRepos(client *http.Client, hostname string, limit int, workspace string, filter FilterOptions) (*RepositoryList, error) {
	// Bitbucket doesn't have the same search capabilities as GitHub
	// Fall back to listing with filters
	result, err := listRepos(client, hostname, limit, workspace, filter)
	if err != nil {
		return nil, err
	}
	result.FromSearch = true
	return result, nil
}
