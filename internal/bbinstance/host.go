// Package bbinstance provides Bitbucket Cloud instance configuration.
package bbinstance

import (
	"errors"
	"fmt"
	"strings"
)

// DefaultHostname is the domain name of Bitbucket Cloud.
const defaultHostname = "bitbucket.org"

// Default returns the host name of Bitbucket Cloud.
func Default() string {
	return defaultHostname
}

// NormalizeHostname normalizes the hostname to lowercase.
// For Bitbucket, we also strip common prefixes like "api." and "www.".
func NormalizeHostname(hostname string) string {
	hostname = strings.ToLower(hostname)
	hostname = strings.TrimPrefix(hostname, "api.")
	hostname = strings.TrimPrefix(hostname, "www.")
	return hostname
}

// IsCloud returns true if the hostname is Bitbucket Cloud.
func IsCloud(hostname string) bool {
	return strings.EqualFold(hostname, defaultHostname)
}

// HostnameValidator validates a Bitbucket hostname.
func HostnameValidator(hostname string) error {
	if len(strings.TrimSpace(hostname)) < 1 {
		return errors.New("a value is required")
	}
	if strings.ContainsRune(hostname, '/') || strings.ContainsRune(hostname, ':') {
		return errors.New("invalid hostname")
	}
	return nil
}

// RESTPrefix returns the REST API base URL for the given hostname.
// For Bitbucket Cloud, this is https://api.bitbucket.org/2.0/
func RESTPrefix(hostname string) string {
	if IsCloud(hostname) {
		return "https://api.bitbucket.org/2.0/"
	}
	// For future Data Center support, would be:
	// return fmt.Sprintf("https://%s/rest/api/1.0/", hostname)
	return fmt.Sprintf("https://api.%s/2.0/", hostname)
}

// HostPrefix returns the web URL prefix for the given hostname.
func HostPrefix(hostname string) string {
	return fmt.Sprintf("https://%s/", hostname)
}

// RepoURL returns the full URL for a repository.
func RepoURL(hostname, workspace, repoSlug string) string {
	return fmt.Sprintf("https://%s/%s/%s", hostname, workspace, repoSlug)
}

// CloneURL returns the HTTPS clone URL for a repository.
func CloneURL(hostname, workspace, repoSlug string) string {
	return fmt.Sprintf("https://%s/%s/%s.git", hostname, workspace, repoSlug)
}

// AuthenticatedCloneURL returns the HTTPS clone URL with credentials embedded.
func AuthenticatedCloneURL(hostname, workspace, repoSlug, username, appPassword string) string {
	return fmt.Sprintf("https://%s:%s@%s/%s/%s.git", username, appPassword, hostname, workspace, repoSlug)
}

// PullRequestURL returns the URL to view a pull request.
func PullRequestURL(hostname, workspace, repoSlug string, prID int) string {
	return fmt.Sprintf("https://%s/%s/%s/pull-requests/%d", hostname, workspace, repoSlug, prID)
}

// IssueURL returns the URL to view an issue.
func IssueURL(hostname, workspace, repoSlug string, issueID int) string {
	return fmt.Sprintf("https://%s/%s/%s/issues/%d", hostname, workspace, repoSlug, issueID)
}
