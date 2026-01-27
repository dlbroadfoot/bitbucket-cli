// Package authflow provides authentication flows for Bitbucket.
// Bitbucket supports two authentication methods:
// 1. API Tokens (new) - requires Atlassian account email + API token
// 2. App Passwords (legacy) - requires Bitbucket username + app password
// OAuth device flow is not supported.
package authflow

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
)

// AuthFlowResult contains the result of an authentication flow.
type AuthFlowResult struct {
	Username string
	Token    string // This is "email:api_token" or "username:app_password" for Bitbucket
}

// APITokenAuth performs authentication using a Bitbucket API Token.
// This is the recommended authentication method for Bitbucket Cloud.
// API tokens require Atlassian account email (not username) for Basic Auth.
func APITokenAuth(hostname string, IO *iostreams.IOStreams, prompter Prompter) (*AuthFlowResult, error) {
	w := IO.ErrOut
	cs := IO.ColorScheme()

	// Show guidance for creating an API Token
	fmt.Fprint(w, `
Tip: you can generate an API Token here https://bitbucket.org/account/settings/api-tokens/
Required scopes: read:account, read:repository, write:repository, read:pullrequest, write:pullrequest

Note: API tokens require your Atlassian account email (not your Bitbucket username).

`)

	// Prompt for email
	email, err := prompter.Input("Atlassian account email:", "")
	if err != nil {
		return nil, err
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Prompt for API token
	apiToken, err := prompter.Password("API token:")
	if err != nil {
		return nil, err
	}
	if apiToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	// Verify credentials and get username
	username, err := verifyCredentialsAndGetUsername(hostname, email, apiToken)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Fprintf(w, "%s Authentication complete. Logged in as %s\n", cs.SuccessIcon(), username)

	// Return the combined token (email:api_token format)
	token := fmt.Sprintf("%s:%s", email, apiToken)

	return &AuthFlowResult{
		Username: username,
		Token:    token,
	}, nil
}

// AppPasswordAuth performs authentication using a Bitbucket App Password (legacy).
// Consider using APITokenAuth instead for new integrations.
func AppPasswordAuth(hostname string, IO *iostreams.IOStreams, prompter Prompter) (*AuthFlowResult, error) {
	w := IO.ErrOut
	cs := IO.ColorScheme()

	// Show guidance for creating an App Password
	fmt.Fprint(w, fmt.Sprintf(`
Tip: you can generate an App Password here https://%s/account/settings/app-passwords/
Required permissions: Account (Read), Repositories (Read, Write), Pull Requests (Read, Write)

`, hostname))

	// Prompt for username
	username, err := prompter.Input("Bitbucket username:", "")
	if err != nil {
		return nil, err
	}
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	// Prompt for app password
	appPassword, err := prompter.Password("App password:")
	if err != nil {
		return nil, err
	}
	if appPassword == "" {
		return nil, fmt.Errorf("app password is required")
	}

	// Verify credentials
	if err := verifyCredentials(hostname, username, appPassword); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Fprintf(w, "%s Authentication complete.\n", cs.SuccessIcon())

	// Return the combined token (username:app_password format)
	token := fmt.Sprintf("%s:%s", username, appPassword)

	return &AuthFlowResult{
		Username: username,
		Token:    token,
	}, nil
}

// verifyCredentialsAndGetUsername checks if the email and API token are valid
// and returns the Bitbucket username associated with the account.
func verifyCredentialsAndGetUsername(hostname, email, apiToken string) (string, error) {
	apiURL := bbinstance.RESTPrefix(hostname) + "user"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	// Set Basic Auth header with email:api_token
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + apiToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Get the username from the response
	var user struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return user.Username, nil
}

// verifyCredentials checks if the username and app password are valid.
func verifyCredentials(hostname, username, appPassword string) error {
	apiURL := bbinstance.RESTPrefix(hostname) + "user"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}

	// Set Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + appPassword))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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

// GetCurrentLogin extracts the username from a Bitbucket token.
// Bitbucket tokens are stored as "username:app_password".
func GetCurrentLogin(token string) (string, error) {
	if idx := strings.Index(token, ":"); idx > 0 {
		return token[:idx], nil
	}
	return "", fmt.Errorf("invalid token format")
}

// Prompter interface for authentication prompts.
type Prompter interface {
	Input(prompt, defaultValue string) (string, error)
	Password(prompt string) (string, error)
	Select(prompt, defaultValue string, options []string) (int, error)
}

// waitForEnter waits for the user to press Enter.
func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}

var oauthSuccessPage = `
<!DOCTYPE html>
<html>
<head>
<title>Bitbucket CLI - Authentication Complete</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; text-align: center; }
h1 { color: #0052CC; }
</style>
</head>
<body>
<h1>Authentication Complete</h1>
<p>You have been authenticated. You may close this window.</p>
</body>
</html>
`
