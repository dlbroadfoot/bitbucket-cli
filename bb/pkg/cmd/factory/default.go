package factory

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	xcolor "github.com/cli/go-gh/v2/pkg/x/color"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	bbContext "github.com/dlbroadfoot/bitbucket-cli/context"
	"github.com/dlbroadfoot/bitbucket-cli/git"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/config"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/internal/prompter"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

func New(appVersion string) *cmdutil.Factory {
	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		Config:         configFunc(), // No factory dependencies
		ExecutableName: "bb",
	}

	f.IOStreams = ioStreams(f)                             // Depends on Config
	f.HttpClient = httpClientFunc(f, appVersion)           // Depends on Config, IOStreams, and appVersion
	f.PlainHttpClient = plainHttpClientFunc(f, appVersion) // Depends on IOStreams, and appVersion
	f.GitClient = newGitClient(f)                          // Depends on IOStreams, and Executable
	f.Remotes = remotesFunc(f)                             // Depends on Config, and GitClient
	f.BaseRepo = BaseRepoFunc(f)                           // Depends on Remotes
	f.Prompter = newPrompter(f)                            // Depends on Config and IOStreams
	f.Browser = newBrowser(f)                              // Depends on Config, and IOStreams
	f.Branch = branchFunc(f)                               // Depends on GitClient

	return f
}

// BaseRepoFunc requests a list of Remotes, and selects the first one.
func BaseRepoFunc(f *cmdutil.Factory) func() (bbrepo.Interface, error) {
	return func() (bbrepo.Interface, error) {
		remotes, err := f.Remotes()
		if err != nil {
			return nil, err
		}
		return remotes[0], nil
	}
}

// SmartBaseRepoFunc provides additional behaviour over BaseRepoFunc.
// For Bitbucket, this is simplified compared to GitHub since we don't have GraphQL
// network resolution.
func SmartBaseRepoFunc(f *cmdutil.Factory) func() (bbrepo.Interface, error) {
	return func() (bbrepo.Interface, error) {
		remotes, err := f.Remotes()
		if err != nil {
			return nil, err
		}
		resolvedRepos, err := bbContext.ResolveRemotesToRepos(remotes, "")
		if err != nil {
			return nil, err
		}
		baseRepo, err := resolvedRepos.BaseRepo(f.IOStreams)
		if err != nil {
			return nil, err
		}

		return baseRepo, nil
	}
}

func remotesFunc(f *cmdutil.Factory) func() (bbContext.Remotes, error) {
	rr := &remoteResolver{
		readRemotes: func() (git.RemoteSet, error) {
			return f.GitClient.Remotes(context.Background())
		},
		getConfig: f.Config,
	}
	return rr.Resolver()
}

func httpClientFunc(f *cmdutil.Factory, appVersion string) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		io := f.IOStreams
		cfg, err := f.Config()
		if err != nil {
			return nil, err
		}
		opts := api.HTTPClientOptions{
			Config:      cfg.Authentication(),
			Log:         io.ErrOut,
			LogColorize: io.ColorEnabled(),
			AppVersion:  appVersion,
		}
		client, err := api.NewHTTPClient(opts)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
}

func plainHttpClientFunc(f *cmdutil.Factory, appVersion string) func() (*http.Client, error) {
	return func() (*http.Client, error) {
		io := f.IOStreams
		opts := api.HTTPClientOptions{
			Log:         io.ErrOut,
			LogColorize: io.ColorEnabled(),
			AppVersion:  appVersion,
			// This is required to prevent automatic setting of auth and other headers.
			SkipDefaultHeaders: true,
		}
		client, err := api.NewHTTPClient(opts)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
}

func newGitClient(f *cmdutil.Factory) *git.Client {
	io := f.IOStreams
	bbPath := f.Executable()
	client := &git.Client{
		GhPath: bbPath,
		Stderr: io.ErrOut,
		Stdin:  io.In,
		Stdout: io.Out,
	}
	return client
}

func newBrowser(f *cmdutil.Factory) browser.Browser {
	io := f.IOStreams
	return browser.New("", io.Out, io.ErrOut)
}

func newPrompter(f *cmdutil.Factory) prompter.Prompter {
	editor, _ := cmdutil.DetermineEditor(f.Config)
	io := f.IOStreams
	return prompter.New(editor, io)
}

func configFunc() func() (gh.Config, error) {
	var cachedConfig gh.Config
	var configError error
	return func() (gh.Config, error) {
		if cachedConfig != nil || configError != nil {
			return cachedConfig, configError
		}
		cachedConfig, configError = config.NewConfig()
		return cachedConfig, configError
	}
}

func branchFunc(f *cmdutil.Factory) func() (string, error) {
	return func() (string, error) {
		currentBranch, err := f.GitClient.CurrentBranch(context.Background())
		if err != nil {
			return "", fmt.Errorf("could not determine current branch: %w", err)
		}
		return currentBranch, nil
	}
}

func ioStreams(f *cmdutil.Factory) *iostreams.IOStreams {
	io := iostreams.System()
	cfg, err := f.Config()
	if err != nil {
		return io
	}

	if _, bbPromptDisabled := os.LookupEnv("BB_PROMPT_DISABLED"); bbPromptDisabled {
		io.SetNeverPrompt(true)
	} else if prompt := cfg.Prompt(""); prompt.Value == "disabled" {
		io.SetNeverPrompt(true)
	}

	falseyValues := []string{"false", "0", "no", ""}

	accessiblePrompterValue, accessiblePrompterIsSet := os.LookupEnv("BB_ACCESSIBLE_PROMPTER")
	if accessiblePrompterIsSet {
		if !slices.Contains(falseyValues, accessiblePrompterValue) {
			io.SetAccessiblePrompterEnabled(true)
		}
	} else if prompt := cfg.AccessiblePrompter(""); prompt.Value == "enabled" {
		io.SetAccessiblePrompterEnabled(true)
	}

	bbSpinnerDisabledValue, bbSpinnerDisabledIsSet := os.LookupEnv("BB_SPINNER_DISABLED")
	if bbSpinnerDisabledIsSet {
		if !slices.Contains(falseyValues, bbSpinnerDisabledValue) {
			io.SetSpinnerDisabled(true)
		}
	} else if spinnerDisabled := cfg.Spinner(""); spinnerDisabled.Value == "disabled" {
		io.SetSpinnerDisabled(true)
	}

	// Pager precedence
	// 1. BB_PAGER
	// 2. pager from config
	// 3. PAGER
	if bbPager, bbPagerExists := os.LookupEnv("BB_PAGER"); bbPagerExists {
		io.SetPager(bbPager)
	} else if pager := cfg.Pager(""); pager.Value != "" {
		io.SetPager(pager.Value)
	}

	if bbColorLabels, bbColorLabelsExists := os.LookupEnv("BB_COLOR_LABELS"); bbColorLabelsExists {
		switch bbColorLabels {
		case "", "0", "false", "no":
			io.SetColorLabels(false)
		default:
			io.SetColorLabels(true)
		}
	} else if prompt := cfg.ColorLabels(""); prompt.Value == "enabled" {
		io.SetColorLabels(true)
	}

	io.SetAccessibleColorsEnabled(xcolor.IsAccessibleColorsEnabled())

	return io
}

// NewCachedHTTPClient wraps an HTTP client with caching.
func NewCachedHTTPClient(client *http.Client, ttl time.Duration) *http.Client {
	// For now, just return the client as-is since we removed the caching infrastructure
	return client
}
