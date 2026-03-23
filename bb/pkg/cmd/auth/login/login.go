package login

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/auth/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

const tokenURL = "https://id.atlassian.com/manage-profile/security/api-tokens"

type LoginOptions struct {
	IO       *iostreams.IOStreams
	Config   func() (gh.Config, error)
	Prompter shared.Prompt
	Browser  browser.Browser

	Interactive bool

	Hostname        string
	Email           string // Atlassian account email for API tokens
	Token           string
	GitProtocol     string
	Web             bool
	InsecureStorage bool
}

func NewCmdLogin(f *cmdutil.Factory, runF func(*LoginOptions) error) *cobra.Command {
	opts := &LoginOptions{
		IO:       f.IOStreams,
		Config:   f.Config,
		Prompter: f.Prompter,
		Browser:  f.Browser,
	}

	var tokenStdin bool

	cmd := &cobra.Command{
		Use:   "login",
		Args:  cobra.ExactArgs(0),
		Short: "Log in to a Bitbucket account",
		Long: heredoc.Docf(`
			Authenticate with Bitbucket Cloud.

			The default hostname is %[1]sbitbucket.org%[1]s.

			Authentication requires a Bitbucket API Token. Use %[1]s--web%[1]s to open the
			Atlassian token creation page in your browser with step-by-step guidance.

			Alternatively, create a token manually at:
			https://id.atlassian.com/manage-profile/security/api-tokens

			Note: API tokens require your Atlassian account email (not your Bitbucket username)
			for authentication.

			Use %[1]s--with-token%[1]s to pass an API token on standard input, or enter it
			interactively when prompted.

			Set the %[1]sBB_TOKEN%[1]s environment variable for automation.

			The git protocol to use for git operations can be set with %[1]s--git-protocol%[1]s.
		`, "`"),
		Example: heredoc.Doc(`
			# Start interactive setup with browser guidance
			$ bb auth login --web

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
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open browser to create token, then prompt for credentials")
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

	// Block login when BB_TOKEN is set — credentials are externally managed
	if token := os.Getenv("BB_TOKEN"); token != "" {
		fmt.Fprintf(opts.IO.ErrOut, "The BB_TOKEN environment variable is set. Credentials are externally managed.\n")
		fmt.Fprintf(opts.IO.ErrOut, "To use 'bb auth login', first unset BB_TOKEN.\n")
		return cmdutil.SilentError
	}

	// Check if token is already set via other environment variables (GH_TOKEN, etc.)
	if src, writeable := shared.AuthTokenWriteable(authCfg, hostname); !writeable {
		fmt.Fprintf(opts.IO.ErrOut, "The value of the %s environment variable is being used for authentication.\n", src)
		fmt.Fprint(opts.IO.ErrOut, "To have Bitbucket CLI store credentials instead, first clear the value from the environment.\n")
		return cmdutil.SilentError
	}

	email := opts.Email
	token := opts.Token

	// Open browser to token creation page if --web is passed
	if opts.Web && opts.Interactive {
		fmt.Fprintln(opts.IO.ErrOut)
		fmt.Fprintln(opts.IO.ErrOut, "Opening Atlassian to create a Bitbucket API token...")
		fmt.Fprintln(opts.IO.ErrOut)
		fmt.Fprintln(opts.IO.ErrOut, "IMPORTANT: Click \"Create API token with scopes\" and select \"Bitbucket\" as the application.")
		fmt.Fprintln(opts.IO.ErrOut)
		fmt.Fprintln(opts.IO.ErrOut, "Required scopes:")
		fmt.Fprintln(opts.IO.ErrOut, "  - User: Read (required for login)")
		fmt.Fprintln(opts.IO.ErrOut, "  - Repositories: Read, Write")
		fmt.Fprintln(opts.IO.ErrOut, "  - Pull requests: Read, Write")
		fmt.Fprintln(opts.IO.ErrOut, "  - Issues: Read, Write (if using issue commands)")
		fmt.Fprintln(opts.IO.ErrOut, "  - Pipelines: Read, Write (if using pipeline commands)")
		fmt.Fprintln(opts.IO.ErrOut)
		if err := opts.Browser.Browse(tokenURL); err != nil {
			fmt.Fprintf(opts.IO.ErrOut, "Failed to open browser: %v\n", err)
			fmt.Fprintf(opts.IO.ErrOut, "Please open %s manually.\n", tokenURL)
		}
		fmt.Fprintln(opts.IO.ErrOut)
	}

	// Interactive prompts
	if opts.Interactive {
		if email == "" {
			if !opts.Web {
				fmt.Fprintln(opts.IO.ErrOut)
				fmt.Fprintln(opts.IO.ErrOut, "Tip: use --web to open the token creation page in your browser")
				fmt.Fprintln(opts.IO.ErrOut)
			}

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

	insecureStorageUsed, loginErr := authCfg.Login(hostname, username, combinedToken, gitProtocol, !opts.InsecureStorage)
	if loginErr != nil {
		return loginErr
	}

	if insecureStorageUsed && !opts.InsecureStorage {
		fmt.Fprintf(opts.IO.ErrOut, "%s Keyring unavailable — credentials saved in plain text\n", cs.Yellow("!"))
		fmt.Fprintf(opts.IO.ErrOut, "  Use --insecure-storage to suppress this warning, or set BB_TOKEN for headless environments\n")
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
	if resp.StatusCode == 403 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("your API token is missing required scopes.\n"+
			"  The token must include User: Read (read:user:bitbucket) scope.\n"+
			"  Re-create the token with the correct scopes using: bb auth login --web\n"+
			"  API response: %s", string(body))
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
