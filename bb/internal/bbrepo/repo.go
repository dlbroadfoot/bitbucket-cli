// Package bbrepo provides Bitbucket repository representation.
package bbrepo

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/bb/v2/internal/bbinstance"
)

// Interface describes an object that represents a Bitbucket repository.
// Note: Bitbucket uses "workspace" (like GitHub's "owner") and "repo_slug" (like GitHub's "repo").
type Interface interface {
	RepoSlug() string
	RepoWorkspace() string
	RepoHost() string
}

// New instantiates a Bitbucket repository from workspace and repo slug arguments.
func New(workspace, repoSlug string) Interface {
	return NewWithHost(workspace, repoSlug, bbinstance.Default())
}

// NewWithHost is like New with an explicit host name.
func NewWithHost(workspace, repoSlug, hostname string) Interface {
	return &bbRepo{
		workspace: workspace,
		slug:      repoSlug,
		hostname:  normalizeHostname(hostname),
	}
}

// FullName serializes a Bitbucket repository into a "WORKSPACE/REPO_SLUG" string.
func FullName(r Interface) string {
	return fmt.Sprintf("%s/%s", r.RepoWorkspace(), r.RepoSlug())
}

// FromFullName extracts the Bitbucket repository information from the following
// formats: "WORKSPACE/REPO_SLUG", "HOST/WORKSPACE/REPO_SLUG", and a full URL.
func FromFullName(nwo string) (Interface, error) {
	return FromFullNameWithHost(nwo, bbinstance.Default())
}

// FromFullNameWithHost is like FromFullName that defaults to a specific host for values that don't
// explicitly include a hostname.
func FromFullNameWithHost(nwo, fallbackHost string) (Interface, error) {
	if strings.HasPrefix(nwo, "https://") || strings.HasPrefix(nwo, "http://") {
		u, err := url.Parse(nwo)
		if err != nil {
			return nil, err
		}
		return FromURL(u)
	}

	parts := strings.SplitN(nwo, "/", 4)
	switch len(parts) {
	case 2:
		// WORKSPACE/REPO_SLUG
		return NewWithHost(parts[0], parts[1], fallbackHost), nil
	case 3:
		// HOST/WORKSPACE/REPO_SLUG
		return NewWithHost(parts[1], parts[2], parts[0]), nil
	default:
		return nil, fmt.Errorf("invalid repository format: %s (expected WORKSPACE/REPO_SLUG)", nwo)
	}
}

// FromURL extracts the Bitbucket repository information from a git remote URL.
func FromURL(u *url.URL) (Interface, error) {
	if u.Hostname() == "" {
		return nil, fmt.Errorf("no hostname detected")
	}

	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid path: %s", u.Path)
	}

	return NewWithHost(parts[0], strings.TrimSuffix(parts[1], ".git"), u.Hostname()), nil
}

func normalizeHostname(h string) string {
	return strings.ToLower(strings.TrimPrefix(h, "www."))
}

// IsSame compares two Bitbucket repositories.
func IsSame(a, b Interface) bool {
	return strings.EqualFold(a.RepoWorkspace(), b.RepoWorkspace()) &&
		strings.EqualFold(a.RepoSlug(), b.RepoSlug()) &&
		normalizeHostname(a.RepoHost()) == normalizeHostname(b.RepoHost())
}

// GenerateRepoURL generates a URL for the repository with an optional path.
func GenerateRepoURL(repo Interface, p string, args ...interface{}) string {
	baseURL := fmt.Sprintf("%s%s/%s", bbinstance.HostPrefix(repo.RepoHost()), repo.RepoWorkspace(), repo.RepoSlug())
	if p != "" {
		if path := fmt.Sprintf(p, args...); path != "" {
			return baseURL + "/" + path
		}
	}
	return baseURL
}

// FormatRemoteURL formats a clone URL for the repository.
func FormatRemoteURL(repo Interface, protocol string) string {
	if protocol == "ssh" {
		return fmt.Sprintf("git@%s:%s/%s.git", repo.RepoHost(), repo.RepoWorkspace(), repo.RepoSlug())
	}
	return fmt.Sprintf("%s%s/%s.git", bbinstance.HostPrefix(repo.RepoHost()), repo.RepoWorkspace(), repo.RepoSlug())
}

// FormatAuthenticatedRemoteURL formats a clone URL with embedded credentials.
func FormatAuthenticatedRemoteURL(repo Interface, username, appPassword string) string {
	return fmt.Sprintf("https://%s:%s@%s/%s/%s.git",
		username, appPassword, repo.RepoHost(), repo.RepoWorkspace(), repo.RepoSlug())
}

type bbRepo struct {
	workspace string
	slug      string
	hostname  string
}

func (r bbRepo) RepoWorkspace() string {
	return r.workspace
}

func (r bbRepo) RepoSlug() string {
	return r.slug
}

func (r bbRepo) RepoHost() string {
	return r.hostname
}
