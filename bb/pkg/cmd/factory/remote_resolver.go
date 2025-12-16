package factory

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dlbroadfoot/bitbucket-cli/context"
	"github.com/dlbroadfoot/bitbucket-cli/git"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/set"
	"github.com/cli/go-gh/v2/pkg/ssh"
)

const (
	BB_HOST = "BB_HOST"
)

type remoteResolver struct {
	readRemotes   func() (git.RemoteSet, error)
	getConfig     func() (gh.Config, error)
	urlTranslator context.Translator
	cachedRemotes context.Remotes
	remotesError  error
}

func (rr *remoteResolver) Resolver() func() (context.Remotes, error) {
	return func() (context.Remotes, error) {
		if rr.cachedRemotes != nil || rr.remotesError != nil {
			return rr.cachedRemotes, rr.remotesError
		}

		gitRemotes, err := rr.readRemotes()
		if err != nil {
			rr.remotesError = err
			return nil, err
		}
		if len(gitRemotes) == 0 {
			rr.remotesError = errors.New("no git remotes found")
			return nil, rr.remotesError
		}

		sshTranslate := rr.urlTranslator
		if sshTranslate == nil {
			sshTranslate = ssh.NewTranslator()
		}
		resolvedRemotes := context.TranslateRemotes(gitRemotes, sshTranslate)

		cfg, err := rr.getConfig()
		if err != nil {
			return nil, err
		}

		authedHosts := cfg.Authentication().Hosts()
		if len(authedHosts) == 0 {
			return nil, errors.New("could not find any host configurations")
		}
		defaultHost, src := cfg.Authentication().DefaultHost()

		// Use set to dedupe list of hosts
		hostsSet := set.NewStringSet()
		hostsSet.AddValues(authedHosts)
		hostsSet.AddValues([]string{defaultHost, bbinstance.Default()})
		hosts := hostsSet.ToSlice()

		// Sort remotes
		sort.Sort(resolvedRemotes)

		rr.cachedRemotes = resolvedRemotes.FilterByHosts(hosts)

		// Filter again by default host if one is set
		// For config file default host fallback to cachedRemotes if none match
		// For environment default host (GH_HOST) do not fallback to cachedRemotes if none match
		if src != "default" {
			filteredRemotes := rr.cachedRemotes.FilterByHosts([]string{defaultHost})
			if isHostEnv(src) || len(filteredRemotes) > 0 {
				rr.cachedRemotes = filteredRemotes
			}
		}

		if len(rr.cachedRemotes) == 0 {
			if isHostEnv(src) {
				rr.remotesError = fmt.Errorf("none of the git remotes configured for this repository correspond to the %s environment variable. Try adding a matching remote or unsetting the variable", src)
				return nil, rr.remotesError
			} else if cfg.Authentication().HasEnvToken() {
				rr.remotesError = errors.New("set the BB_HOST environment variable to specify which Bitbucket host to use")
				return nil, rr.remotesError
			}
			rr.remotesError = errors.New("none of the git remotes configured for this repository point to a known Bitbucket host. To tell bb about a new Bitbucket host, please use `bb auth login`")
			return nil, rr.remotesError
		}

		return rr.cachedRemotes, nil
	}
}

func isHostEnv(src string) bool {
	return src == BB_HOST
}
