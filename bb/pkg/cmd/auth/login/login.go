package login

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/auth/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type LoginOptions struct {
	IO       *iostreams.IOStreams
	Config   func() (gh.Config, error)
	Prompter shared.Prompt

	Interactive bool

	Hostname        string
	Email           string // Atlassian account email for API tokens
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

			Authentication requires a Bitbucket API Token. To create one:
			1. Go to https://bitbucket.org/account/settings/api-tokens/
			2. Click "Create API token with scopes"
			3. Select scopes: read:account, read:repository, write:repository, read:pullrequest, write:pullrequest
			4. Copy the generated token

			Note: API tokens require your Atlassian account email (not your Bitbucket username)
			for authentication.

			Use %[1]s--with-token%[1]s to pass an API token on standard input, or enter it
			interactively when prompted.

			Alternatively, set the %[1]sBB_TOKEN%[1]s environment variable with your API token.
			This method is most suitable for automation. See %[1]sbb help environment%[1]s for more info.

			The git protocol to use for git operations can be set with %[1]s--git-protocol%[1]s.
		`, "`"),
		Example: heredoc.Doc(`
			# Start interactive setup
			$ bb auth login

			# Authenticate by reading the API token from a file
			$ bb auth login --with-token < mytoken.txt

			# Authenticate with a specific email
			$ bb auth login --email user@example.com
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
	cmd.Flags().StringVarP(&opts.Email, "email", "e", "", "Atlassian account email for API token authentication")
	cmd.Flags().BoolVar(&tokenStdin, "with-token", false, "Read API token from standard input")
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

	email := opts.Email
	token := opts.Token

	// Interactive prompts
	if opts.Interactive {
		if email == "" {
			fmt.Fprintln(opts.IO.ErrOut)
			fmt.Fprintln(opts.IO.ErrOut, "Tip: Create an API token at https://bitbucket.org/account/settings/api-tokens/")
			fmt.Fprintln(opts.IO.ErrOut, "Required scopes: read:account, read:repository, write:repository, read:pullrequest, write:pullrequest")
			fmt.Fprintln(opts.IO.ErrOut)

			var err error
			email, err = opts.Prompter.Input("Atlassian account email:", "")
			if err != nil {
				return err
			}
			email = strings.TrimSpace(email)
		}

		if token == "" {
			var err error
			token, err = opts.Prompter.Password("API token:")
			if err != nil {
				return err
			}
		}
	}

	if email == "" {
		return fmt.Errorf("email is required (use --email or enter interactively)")
	}
	if token == "" {
		return fmt.Errorf("API token is required (use --with-token or enter interactively)")
	}

	// Verify credentials and get username
	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.ErrOut, "%s Verifying credentials...\n", cs.Yellow("!"))

	username, err := verifyCredentialsAndGetUsername(hostname, email, token)
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
	// For Bitbucket API tokens, we store email:token for Basic Auth
	combinedToken := email + ":" + token

	_, loginErr := authCfg.Login(hostname, username, combinedToken, gitProtocol, !opts.InsecureStorage)
	if loginErr != nil {
		return loginErr
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s Logged in as %s\n", cs.SuccessIcon(), cs.Bold(username))

	return nil
}

// verifyCredentialsAndGetUsername checks if the email and API token are valid
// and returns the Bitbucket username associated with the account.
// It uses the /user endpoint which requires read:account scope.
func verifyCredentialsAndGetUsername(hostname, email, token string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", bbinstance.RESTPrefix(hostname)+"user", nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("invalid email or API token")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected response status: %d: %s", resp.StatusCode, string(body))
	}

	// Parse the username from the user response
	var userResp struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", fmt.Errorf("failed to parse user response: %w", err)
	}

	if userResp.Username == "" {
		return "", fmt.Errorf("no username in response")
	}

	return userResp.Username, nil
}
