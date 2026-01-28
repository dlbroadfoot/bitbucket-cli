package refresh

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/git"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/auth/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/auth/shared/gitcredentials"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type RefreshOptions struct {
	IO        *iostreams.IOStreams
	Config    func() (gh.Config, error)
	GitClient *git.Client
	Prompter  shared.Prompt

	MainExecutable string

	Hostname string

	Interactive     bool
	InsecureStorage bool
}

func NewCmdRefresh(f *cmdutil.Factory, runF func(*RefreshOptions) error) *cobra.Command {
	opts := &RefreshOptions{
		IO:        f.IOStreams,
		Config:    f.Config,
		GitClient: f.GitClient,
		Prompter:  f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Args:  cobra.ExactArgs(0),
		Short: "Refresh stored authentication credentials",
		Long: heredoc.Docf(`
			Re-authenticate with Bitbucket.

			This command will prompt you to re-enter your API token credentials.

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
	cmd.Flags().BoolVarP(&opts.InsecureStorage, "insecure-storage", "", false, "Save authentication credentials in plain text instead of credential store")

	return cmd
}

func refreshRun(opts *RefreshOptions) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}
	authCfg := cfg.Authentication()

	candidates := authCfg.Hosts()
	if len(candidates) == 0 {
		return fmt.Errorf("not logged in to any hosts. Use 'bb auth login' to authenticate with a host")
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
			return fmt.Errorf("not logged in to %s. use 'bb auth login' to authenticate with this host", hostname)
		}
	}

	if src, writeable := shared.AuthTokenWriteable(authCfg, hostname); !writeable {
		fmt.Fprintf(opts.IO.ErrOut, "The value of the %s environment variable is being used for authentication.\n", src)
		fmt.Fprint(opts.IO.ErrOut, "To refresh credentials stored in Bitbucket CLI, first clear the value from the environment.\n")
		return cmdutil.SilentError
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
	}

	// Prompt for credentials
	cs := opts.IO.ColorScheme()

	fmt.Fprint(opts.IO.ErrOut, `
Tip: you can generate an API token here https://id.atlassian.com/manage-profile/security/api-tokens
Required scopes: read:user, read:account, read:repository, write:repository, read:pullrequest, write:pullrequest

`)

	username, err := opts.Prompter.Input("Atlassian account email:", "")
	if err != nil {
		return err
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}

	appPassword, err := opts.Prompter.Password("API token:")
	if err != nil {
		return err
	}
	if appPassword == "" {
		return fmt.Errorf("API token is required")
	}

	// Verify credentials
	fmt.Fprintf(opts.IO.ErrOut, "%s Verifying credentials...\n", cs.Yellow("!"))
	if err := verifyCredentials(hostname, username, appPassword); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	activeUser, _ := authCfg.ActiveUser(hostname)
	if activeUser != "" && activeUser != username {
		return fmt.Errorf("error refreshing credentials for %s, received credentials for %s, did you use the correct account?", activeUser, username)
	}

	// Store combined token (email:api_token format)
	combinedToken := username + ":" + appPassword
	if _, err := authCfg.Login(hostname, username, combinedToken, "", !opts.InsecureStorage); err != nil {
		return err
	}

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

// verifyCredentials checks if the email and API token are valid
func verifyCredentials(hostname, username, token string) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", bbinstance.RESTPrefix(hostname)+"user", nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(username, token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid email or API token")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return nil
}
