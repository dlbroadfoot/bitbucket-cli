// Package authflow provides authentication flows for Bitbucket.
// Bitbucket uses App Passwords (similar to GitHub PATs) for authentication.
// OAuth device flow is not supported - we use simple username:app_password authentication.
package authflow

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cli/bb/v2/internal/bbinstance"
	"github.com/cli/bb/v2/pkg/iostreams"
)

// AuthFlowResult contains the result of an authentication flow.
type AuthFlowResult struct {
	Username string
	Token    string // This is "username:app_password" for Bitbucket
}

// AppPasswordAuth performs authentication using a Bitbucket App Password.
// This is the primary authentication method for Bitbucket (no OAuth device flow).
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
