package view

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/cli/go-gh/v2/pkg/asciisanitizer"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"golang.org/x/text/transform"
)

var NotFoundError = errors.New("not found")

type RepoReadme struct {
	Filename string
	Content  string
	BaseURL  string
}

// RepositoryReadme fetches the README for a Bitbucket repository.
// Bitbucket API: GET /2.0/repositories/{workspace}/{repo_slug}/src/{commit}/README.md
func RepositoryReadme(client *http.Client, repo bbrepo.Interface, branch string) (*RepoReadme, error) {
	apiClient := api.NewClientFromHTTP(client)

	// First, try to get the README from the repository source
	// Bitbucket doesn't have a dedicated README endpoint, so we need to fetch from src
	ref := branch
	if ref == "" {
		ref = "HEAD"
	}

	// Try common README filenames
	readmeNames := []string{"README.md", "readme.md", "README.markdown", "README", "readme"}

	for _, filename := range readmeNames {
		path := fmt.Sprintf("repositories/%s/%s/src/%s/%s",
			repo.RepoWorkspace(), repo.RepoSlug(), url.PathEscape(ref), filename)

		req, err := http.NewRequest("GET", api.RESTPrefix(repo.RepoHost())+path, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, api.HandleHTTPError(resp)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		sanitized, err := io.ReadAll(transform.NewReader(bytes.NewReader(content), &asciisanitizer.Sanitizer{}))
		if err != nil {
			return nil, err
		}

		// Build the HTML URL for the README
		baseURL := bbrepo.GenerateRepoURL(repo, "src/%s/%s", url.PathEscape(ref), filename)

		return &RepoReadme{
			Filename: filename,
			Content:  string(sanitized),
			BaseURL:  baseURL,
		}, nil
	}

	// If we couldn't find any README, check if the repo has a description as fallback
	var repoInfo struct {
		Description string `json:"description"`
	}
	repoPath := fmt.Sprintf("repositories/%s/%s", repo.RepoWorkspace(), repo.RepoSlug())
	if err := apiClient.Get(repo.RepoHost(), repoPath, &repoInfo); err == nil && repoInfo.Description != "" {
		return nil, NotFoundError
	}

	return nil, NotFoundError
}

// RESTPrefix returns the REST API base URL for the given hostname.
// This is a helper to avoid import cycle with bbinstance.
func init() {
	// Use the api package's method
}
