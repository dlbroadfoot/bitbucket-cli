// Package context provides repository resolution from git remotes.
package context

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

func ResolveRemotesToRepos(remotes Remotes, base string) (*ResolvedRemotes, error) {
	sort.Stable(remotes)

	result := &ResolvedRemotes{
		remotes: remotes,
	}

	var baseOverride bbrepo.Interface
	if base != "" {
		var err error
		baseOverride, err = bbrepo.FromFullName(base)
		if err != nil {
			return result, err
		}
		result.baseOverride = baseOverride
	}

	return result, nil
}

type ResolvedRemotes struct {
	baseOverride bbrepo.Interface
	remotes      Remotes
}

func (r *ResolvedRemotes) BaseRepo(io *iostreams.IOStreams) (bbrepo.Interface, error) {
	if r.baseOverride != nil {
		return r.baseOverride, nil
	}

	if len(r.remotes) == 0 {
		return nil, errors.New("no git remotes")
	}

	// if any of the remotes already has a resolution, respect that
	for _, r := range r.remotes {
		if r.Resolved == "base" {
			return r, nil
		} else if r.Resolved != "" {
			repo, err := bbrepo.FromFullName(r.Resolved)
			if err != nil {
				return nil, err
			}
			return bbrepo.NewWithHost(repo.RepoWorkspace(), repo.RepoSlug(), r.RepoHost()), nil
		}
	}

	if !io.CanPrompt() {
		// we cannot prompt, so just resort to the 1st remote
		return r.remotes[0], nil
	}

	// For Bitbucket, we don't have a repo network query, so just use the first remote
	// or prompt if multiple
	if len(r.remotes) == 1 {
		return r.remotes[0], nil
	}

	cs := io.ColorScheme()

	fmt.Fprintf(io.ErrOut,
		"%s No default remote repository has been set. To learn more about the default repository, run: bb repo set-default --help\n",
		cs.FailureIcon())

	fmt.Fprintln(io.Out)

	return nil, errors.New(
		"please run `bb repo set-default` to select a default remote repository.")
}

// RemoteForRepo finds the git remote that points to a repository
func (r *ResolvedRemotes) RemoteForRepo(repo bbrepo.Interface) (*Remote, error) {
	for _, remote := range r.remotes {
		if bbrepo.IsSame(remote, repo) {
			return remote, nil
		}
	}
	return nil, errors.New("not found")
}

// Remotes returns the list of remotes
func (r *ResolvedRemotes) Remotes() Remotes {
	return r.remotes
}
