package shared

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

type iconfig interface {
	Login(string, string, string, string, bool) (bool, error)
	UsersForHost(string) []string
}

type LoginOptions struct {
	IO             *iostreams.IOStreams
	Config         iconfig
	HTTPClient     *http.Client
	Hostname       string
	Interactive    bool
	GitProtocol    string
	Prompter       Prompt
	CredentialFlow *GitCredentialFlow
	SecureStorage  bool
}

// Login performs the Bitbucket login flow using App Passwords.
// App Passwords are Bitbucket's equivalent of GitHub PATs.
func Login(opts *LoginOptions) error {
	cfg := opts.Config
	hostname := opts.Hostname
	cs := opts.IO.ColorScheme()

	gitProtocol := strings.ToLower(opts.GitProtocol)
	if opts.Interactive && gitProtocol == "" {
		options := []string{
			"HTTPS",
			"SSH",
		}
		result, err := opts.Prompter.Select(
			"What is your preferred protocol for Git operations on this host?",
			options[0],
			options)
		if err != nil {
			return err
		}
		proto := options[result]
		gitProtocol = strings.ToLower(proto)
	}

	if opts.Interactive && gitProtocol == "https" {
		if err := opts.CredentialFlow.Prompt(hostname); err != nil {
			return err
		}
	}

	// Bitbucket uses App Passwords - prompt for username and app password
	fmt.Fprint(opts.IO.ErrOut, heredoc.Docf(`
		Tip: you can generate an App Password here https://%s/account/settings/app-passwords/
		Required permissions: Account (Read), Repositories (Read, Write), Pull Requests (Read, Write)
	`, hostname))

	username, err := opts.Prompter.Input("Bitbucket username:", "")
	if err != nil {
		return err
	}
	if username == "" {
		return fmt.Errorf("username is required")
	}

	appPassword, err := opts.Prompter.Password("App password:")
	if err != nil {
		return err
	}
	if appPassword == "" {
		return fmt.Errorf("app password is required")
	}

	// Bitbucket tokens are stored as "username:app_password"
	authToken := fmt.Sprintf("%s:%s", username, appPassword)

	// Verify the credentials by calling the Bitbucket API
	if err := verifyCredentials(opts.HTTPClient, hostname, username, appPassword); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s Authentication complete.\n", cs.SuccessIcon())

	// Get these users before adding the new one, so that we can
	// check whether the user was already logged in later.
	usersForHost := cfg.UsersForHost(hostname)
	userWasAlreadyLoggedIn := slices.Contains(usersForHost, username)

	if gitProtocol != "" {
		fmt.Fprintf(opts.IO.ErrOut, "- bb config set -h %s git_protocol %s\n", hostname, gitProtocol)
		fmt.Fprintf(opts.IO.ErrOut, "%s Configured git protocol\n", cs.SuccessIcon())
	}

	insecureStorageUsed, err := cfg.Login(hostname, username, authToken, gitProtocol, opts.SecureStorage)
	if err != nil {
		return err
	}
	if insecureStorageUsed {
		fmt.Fprintf(opts.IO.ErrOut, "%s Authentication credentials saved in plain text\n", cs.Yellow("!"))
	}

	if opts.CredentialFlow.ShouldSetup() {
		err := opts.CredentialFlow.Setup(hostname, username, authToken)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(opts.IO.ErrOut, "%s Logged in as %s\n", cs.SuccessIcon(), cs.Bold(username))
	if userWasAlreadyLoggedIn {
		fmt.Fprintf(opts.IO.ErrOut, "%s You were already logged in to this account\n", cs.WarningIcon())
	}

	return nil
}

// verifyCredentials verifies the username and app password against the Bitbucket API.
func verifyCredentials(httpClient *http.Client, hostname, username, appPassword string) error {
	apiURL := bbinstance.RESTPrefix(hostname) + "user"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, appPassword)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	// Verify the username matches
	var user struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !strings.EqualFold(user.Username, username) {
		return fmt.Errorf("username mismatch: expected %s, got %s", username, user.Username)
	}

	return nil
}

// GetCurrentLogin returns the username from a Bitbucket token (which is in username:app_password format).
func GetCurrentLogin(httpClient httpClient, hostname, authToken string) (string, error) {
	// For Bitbucket, the username is the first part of the token
	if idx := strings.Index(authToken, ":"); idx > 0 {
		return authToken[:idx], nil
	}
	return "", fmt.Errorf("invalid token format")
}
