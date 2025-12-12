package refresh

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/git"
	"github.com/cli/bb/v2/internal/authflow"
	"github.com/cli/bb/v2/internal/gh"
	"github.com/cli/bb/v2/pkg/cmd/auth/shared"
	"github.com/cli/bb/v2/pkg/cmd/auth/shared/gitcredentials"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/cli/bb/v2/pkg/set"
	"github.com/spf13/cobra"
)

type token string
type username string

type RefreshOptions struct {
	IO              *iostreams.IOStreams
	Config          func() (gh.Config, error)
	PlainHttpClient func() (*http.Client, error)
	GitClient       *git.Client
	Prompter        shared.Prompt

	MainExecutable string

	Hostname     string
	Scopes       []string
	RemoveScopes []string
	ResetScopes  bool
	AuthFlow     func(*http.Client, *iostreams.IOStreams, string, []string, bool, bool) (token, username, error)

	Interactive     bool
	InsecureStorage bool
	Clipboard       bool
}

func NewCmdRefresh(f *cmdutil.Factory, runF func(*RefreshOptions) error) *cobra.Command {
	opts := &RefreshOptions{
		IO:     f.IOStreams,
		Config: f.Config,
		AuthFlow: func(httpClient *http.Client, io *iostreams.IOStreams, hostname string, scopes []string, interactive bool, clipboard bool) (token, username, error) {
			t, u, err := authflow.AuthFlow(httpClient, hostname, io, "", scopes, interactive, f.Browser, clipboard)
			return token(t), username(u), err
		},
		PlainHttpClient: f.PlainHttpClient,
		GitClient:       f.GitClient,
		Prompter:        f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Args:  cobra.ExactArgs(0),
		Short: "Refresh stored authentication credentials",
		Long: heredoc.Docf(`
			Re-authenticate with Bitbucket.

			This command will prompt you to re-enter your App Password credentials.

			If you have multiple accounts in %[1]sbb auth status%[1]s and want to refresh the credentials for an
			inactive account, you will have to use %[1]sbb auth switch%[1]s to that account first before using
			this command, and then switch back when you are done.
		`, "`"),
		Example: heredoc.Doc(`
			# Re-authenticate with Bitbucket
			$ bb auth refresh

			# Re-authenticate with a specific host
			$ bb auth refresh --hostname bitbucket.org
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Interactive = opts.IO.CanPrompt()

			if !opts.Interactive && opts.Hostname == "" {
				return cmdutil.FlagErrorf("--hostname required when not running interactively")
			}

			opts.MainExecutable = f.Executable()
			if runF != nil {
				return runF(opts)
			}
			return refreshRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "The Bitbucket host to use for authentication")
	cmd.Flags().StringSliceVarP(&opts.Scopes, "scopes", "s", nil, "Additional authentication scopes (not used for App Passwords)")
	cmd.Flags().StringSliceVarP(&opts.RemoveScopes, "remove-scopes", "r", nil, "Authentication scopes to remove (not used for App Passwords)")
	cmd.Flags().BoolVar(&opts.ResetScopes, "reset-scopes", false, "Reset authentication scopes (not used for App Passwords)")
	cmd.Flags().BoolVarP(&opts.Clipboard, "clipboard", "c", false, "Copy one-time code to clipboard (not used for App Passwords)")
	// secure storage became the default on 2023/4/04; this flag is left as a no-op for backwards compatibility
	var secureStorage bool
	cmd.Flags().BoolVar(&secureStorage, "secure-storage", false, "Save authentication credentials in secure credential store")
	_ = cmd.Flags().MarkHidden("secure-storage")

	cmd.Flags().BoolVarP(&opts.InsecureStorage, "insecure-storage", "", false, "Save authentication credentials in plain text instead of credential store")

	return cmd
}

func refreshRun(opts *RefreshOptions) error {
	plainHTTPClient, err := opts.PlainHttpClient()
	if err != nil {
		return err
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}
	authCfg := cfg.Authentication()

	candidates := authCfg.Hosts()
	if len(candidates) == 0 {
		return fmt.Errorf("not logged in to any hosts. Use 'gh auth login' to authenticate with a host")
	}

	hostname := opts.Hostname
	if hostname == "" {
		if len(candidates) == 1 {
			hostname = candidates[0]
		} else {
			selected, err := opts.Prompter.Select("What account do you want to refresh auth for?", "", candidates)
			if err != nil {
				return fmt.Errorf("could not prompt: %w", err)
			}
			hostname = candidates[selected]
		}
	} else {
		var found bool
		for _, c := range candidates {
			if c == hostname {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("not logged in to %s. use 'gh auth login' to authenticate with this host", hostname)
		}
	}

	if src, writeable := shared.AuthTokenWriteable(authCfg, hostname); !writeable {
		fmt.Fprintf(opts.IO.ErrOut, "The value of the %s environment variable is being used for authentication.\n", src)
		fmt.Fprint(opts.IO.ErrOut, "To refresh credentials stored in GitHub CLI, first clear the value from the environment.\n")
		return cmdutil.SilentError
	}

	additionalScopes := set.NewStringSet()

	if !opts.ResetScopes {
		if oldToken, _ := authCfg.ActiveToken(hostname); oldToken != "" {
			if oldScopes, err := shared.GetScopes(plainHTTPClient, hostname, oldToken); err == nil {
				for _, s := range strings.Split(oldScopes, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						additionalScopes.Add(s)
					}
				}
			}
		}
	}

	credentialFlow := &shared.GitCredentialFlow{
		Prompter: opts.Prompter,
		HelperConfig: &gitcredentials.HelperConfig{
			SelfExecutablePath: opts.MainExecutable,
			GitClient:          opts.GitClient,
		},
		Updater: &gitcredentials.Updater{
			GitClient: opts.GitClient,
		},
	}
	gitProtocol := cfg.GitProtocol(hostname).Value
	if opts.Interactive && gitProtocol == "https" {
		if err := credentialFlow.Prompt(hostname); err != nil {
			return err
		}
		additionalScopes.AddValues(credentialFlow.Scopes())
	}

	additionalScopes.AddValues(opts.Scopes)

	additionalScopes.RemoveValues(opts.RemoveScopes)

	authedToken, authedUser, err := opts.AuthFlow(plainHTTPClient, opts.IO, hostname, additionalScopes.ToSlice(), opts.Interactive, opts.Clipboard)
	if err != nil {
		return err
	}
	activeUser, _ := authCfg.ActiveUser(hostname)
	if activeUser != "" && username(activeUser) != authedUser {
		return fmt.Errorf("error refreshing credentials for %s, received credentials for %s, did you use the correct account in the browser?", activeUser, authedUser)
	}
	if _, err := authCfg.Login(hostname, string(authedUser), string(authedToken), "", !opts.InsecureStorage); err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.ErrOut, "%s Authentication complete.\n", cs.SuccessIcon())

	if credentialFlow.ShouldSetup() {
		username, _ := authCfg.ActiveUser(hostname)
		password, _ := authCfg.ActiveToken(hostname)
		if err := credentialFlow.Setup(hostname, username, password); err != nil {
			return err
		}
	}

	return nil
}
