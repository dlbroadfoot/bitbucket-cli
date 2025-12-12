package login

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/internal/bbinstance"
	"github.com/cli/bb/v2/internal/gh"
	"github.com/cli/bb/v2/pkg/cmd/auth/shared"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type LoginOptions struct {
	IO              *iostreams.IOStreams
	Config          func() (gh.Config, error)
	Prompter        shared.Prompt

	Interactive bool

	Hostname        string
	Username        string
	Token           string
	GitProtocol     string
	InsecureStorage bool
}

func NewCmdLogin(f *cmdutil.Factory, runF func(*LoginOptions) error) *cobra.Command {
	opts := &LoginOptions{
		IO:       f.IOStreams,
		Config:   f.Config,
		Prompter: f.Prompter,
	}

	var tokenStdin bool

	cmd := &cobra.Command{
		Use:   "login",
		Args:  cobra.ExactArgs(0),
		Short: "Log in to a Bitbucket account",
		Long: heredoc.Docf(`
			Authenticate with Bitbucket Cloud.

			The default hostname is %[1]sbitbucket.org%[1]s.

			Authentication requires a Bitbucket App Password. To create one:
			1. Go to https://bitbucket.org/account/settings/app-passwords/
			2. Click "Create app password"
			3. Select permissions: Account (Read), Repositories (Read, Write), Pull requests (Read, Write)
			4. Copy the generated password

			Use %[1]s--with-token%[1]s to pass an app password on standard input, or enter it
			interactively when prompted.

			Alternatively, set the %[1]sBB_TOKEN%[1]s environment variable with your app password.
			This method is most suitable for automation. See %[1]sbb help environment%[1]s for more info.

			The git protocol to use for git operations can be set with %[1]s--git-protocol%[1]s.
		`, "`"),
		Example: heredoc.Doc(`
			# Start interactive setup
			$ bb auth login

			# Authenticate by reading the app password from a file
			$ bb auth login --with-token < mytoken.txt

			# Authenticate with a specific username
			$ bb auth login --username myuser
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenStdin {
				defer opts.IO.In.Close()
				token, err := io.ReadAll(opts.IO.In)
				if err != nil {
					return fmt.Errorf("failed to read token from standard input: %w", err)
				}
				opts.Token = strings.TrimSpace(string(token))
			}

			if opts.IO.CanPrompt() && opts.Token == "" {
				opts.Interactive = true
			}

			if cmd.Flags().Changed("hostname") {
				if err := bbinstance.HostnameValidator(opts.Hostname); err != nil {
					return cmdutil.FlagErrorf("error parsing hostname: %w", err)
				}
			}

			if opts.Hostname == "" {
				opts.Hostname = bbinstance.Default()
			}

			if runF != nil {
				return runF(opts)
			}

			return loginRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "The hostname of the Bitbucket instance to authenticate with")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Bitbucket username")
	cmd.Flags().BoolVar(&tokenStdin, "with-token", false, "Read app password from standard input")
	cmdutil.StringEnumFlag(cmd, &opts.GitProtocol, "git-protocol", "p", "", []string{"ssh", "https"}, "The protocol to use for git operations on this host")
	cmd.Flags().BoolVar(&opts.InsecureStorage, "insecure-storage", false, "Save authentication credentials in plain text instead of credential store")

	return cmd
}

func loginRun(opts *LoginOptions) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}
	authCfg := cfg.Authentication()

	hostname := strings.ToLower(opts.Hostname)

	// Check if token is already set via environment
	if src, writeable := shared.AuthTokenWriteable(authCfg, hostname); !writeable {
		fmt.Fprintf(opts.IO.ErrOut, "The value of the %s environment variable is being used for authentication.\n", src)
		fmt.Fprint(opts.IO.ErrOut, "To have Bitbucket CLI store credentials instead, first clear the value from the environment.\n")
		return cmdutil.SilentError
	}

	username := opts.Username
	token := opts.Token

	// Interactive prompts
	if opts.Interactive {
		if username == "" {
			var err error
			username, err = opts.Prompter.Input("Bitbucket username:", "")
			if err != nil {
				return err
			}
			username = strings.TrimSpace(username)
		}

		if token == "" {
			var err error
			token, err = opts.Prompter.Password("App password:")
			if err != nil {
				return err
			}
		}
	}

	if username == "" {
		return fmt.Errorf("username is required")
	}
	if token == "" {
		return fmt.Errorf("app password is required")
	}

	// Verify credentials by making an API call
	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.ErrOut, "%s Verifying credentials...\n", cs.Yellow("!"))

	err = verifyCredentials(hostname, username, token)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Prompt for git protocol if not specified
	gitProtocol := opts.GitProtocol
	if opts.Interactive && gitProtocol == "" {
		options := []string{"HTTPS", "SSH"}
		selected, err := opts.Prompter.Select(
			"What is your preferred protocol for Git operations on this host?",
			options[0],
			options)
		if err != nil {
			return err
		}
		gitProtocol = strings.ToLower(options[selected])
	}

	// Store credentials
	// For Bitbucket, we store both username and token since Basic Auth requires both
	// We'll store them as "username:token" in the token field
	combinedToken := username + ":" + token

	_, loginErr := authCfg.Login(hostname, username, combinedToken, gitProtocol, !opts.InsecureStorage)
	if loginErr != nil {
		return loginErr
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s Logged in as %s\n", cs.SuccessIcon(), cs.Bold(username))

	return nil
}

// verifyCredentials checks if the username and app password are valid
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
		return fmt.Errorf("invalid username or app password")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return nil
}
