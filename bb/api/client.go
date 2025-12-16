package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
)

const (
	accept        = "Accept"
	authorization = "Authorization"
	contentType   = "Content-Type"
	userAgent     = "User-Agent"
)

// Client is a Bitbucket API client.
type Client struct {
	http *http.Client
}

// NewClientFromHTTP creates a new Client from an existing http.Client.
func NewClientFromHTTP(httpClient *http.Client) *Client {
	return &Client{http: httpClient}
}

// HTTP returns the underlying http.Client.
func (c *Client) HTTP() *http.Client {
	return c.http
}

// HTTPError represents an HTTP error response from the Bitbucket API.
type HTTPError struct {
	StatusCode int
	Message    string
	RequestURL *url.URL
	Body       string
}

func (e HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s (%s)", e.StatusCode, e.Message, e.RequestURL)
	}
	return fmt.Sprintf("HTTP %d (%s)", e.StatusCode, e.RequestURL)
}

// REST performs a REST request and parses the response.
func (c *Client) REST(hostname string, method string, path string, body io.Reader, data interface{}) error {
	url := bbinstance.RESTPrefix(hostname) + strings.TrimPrefix(path, "/")
	return c.RESTWithURL(method, url, body, data)
}

// RESTWithURL performs a REST request to a full URL and parses the response.
func (c *Client) RESTWithURL(method string, url string, body io.Reader, data interface{}) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set(accept, "application/json")
	if body != nil {
		req.Header.Set(contentType, "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return HandleHTTPError(resp)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if data == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(data)
}

// RESTWithNext performs a REST request and returns the next page URL from the response.
// This is used for Bitbucket's pagination which includes a "next" field in the response body.
func (c *Client) RESTWithNext(hostname string, method string, path string, body io.Reader, data interface{}) (string, error) {
	url := bbinstance.RESTPrefix(hostname) + strings.TrimPrefix(path, "/")
	return c.RESTWithNextURL(method, url, body, data)
}

// RESTWithNextURL performs a REST request to a full URL and returns the next page URL.
func (c *Client) RESTWithNextURL(method string, url string, body io.Reader, data interface{}) (string, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set(accept, "application/json")
	if body != nil {
		req.Header.Set(contentType, "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return "", HandleHTTPError(resp)
	}

	if resp.StatusCode == http.StatusNoContent {
		return "", nil
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if data != nil {
		if err := json.Unmarshal(responseBody, data); err != nil {
			return "", err
		}
	}

	// Extract "next" URL from paginated response
	var paginationInfo struct {
		Next string `json:"next"`
	}
	if err := json.Unmarshal(responseBody, &paginationInfo); err != nil {
		return "", nil // Ignore pagination parsing errors
	}

	return paginationInfo.Next, nil
}

// RESTWithContext performs a REST request with a context.
func (c *Client) RESTWithContext(ctx context.Context, hostname string, method string, path string, body io.Reader, data interface{}) error {
	url := bbinstance.RESTPrefix(hostname) + strings.TrimPrefix(path, "/")

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set(accept, "application/json")
	if body != nil {
		req.Header.Set(contentType, "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return HandleHTTPError(resp)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if data == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(data)
}

// Delete performs a DELETE request.
func (c *Client) Delete(hostname string, path string) error {
	return c.REST(hostname, http.MethodDelete, path, nil, nil)
}

// Get performs a GET request and parses the response.
func (c *Client) Get(hostname string, path string, data interface{}) error {
	return c.REST(hostname, http.MethodGet, path, nil, data)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(hostname string, path string, input interface{}, data interface{}) error {
	body, err := jsonBody(input)
	if err != nil {
		return err
	}
	return c.REST(hostname, http.MethodPost, path, body, data)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(hostname string, path string, input interface{}, data interface{}) error {
	body, err := jsonBody(input)
	if err != nil {
		return err
	}
	return c.REST(hostname, http.MethodPut, path, body, data)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(hostname string, path string, input interface{}, data interface{}) error {
	body, err := jsonBody(input)
	if err != nil {
		return err
	}
	return c.REST(hostname, http.MethodPatch, path, body, data)
}

// jsonBody encodes input as JSON and returns a reader.
func jsonBody(input interface{}) (io.Reader, error) {
	if input == nil {
		return nil, nil
	}
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// HandleHTTPError parses an HTTP response into an HTTPError.
func HandleHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var message string

	// Try to parse Bitbucket error response
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		message = errResp.Error.Message
		if errResp.Error.Detail != "" {
			message = fmt.Sprintf("%s: %s", message, errResp.Error.Detail)
		}
	} else {
		// Fallback to raw body
		message = strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
	}

	return HTTPError{
		StatusCode: resp.StatusCode,
		Message:    message,
		RequestURL: resp.Request.URL,
		Body:       string(body),
	}
}

// IsNotFoundError checks if an error is a 404 Not Found error.
func IsNotFoundError(err error) bool {
	var httpErr HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound
}

// IsUnauthorizedError checks if an error is a 401 Unauthorized error.
func IsUnauthorizedError(err error) bool {
	var httpErr HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusUnauthorized
}

// IsForbiddenError checks if an error is a 403 Forbidden error.
func IsForbiddenError(err error) bool {
	var httpErr HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden
}

// IsConflictError checks if an error is a 409 Conflict error.
func IsConflictError(err error) bool {
	var httpErr HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusConflict
}

// CurrentLoginName returns the username of the currently authenticated user.
// Bitbucket API: GET /2.0/user
func CurrentLoginName(client *Client, hostname string) (string, error) {
	var user User
	if err := client.Get(hostname, "user", &user); err != nil {
		return "", err
	}
	return user.Username, nil
}

// CurrentUser returns the full user object for the currently authenticated user.
// Bitbucket API: GET /2.0/user
func CurrentUser(client *Client, hostname string) (*User, error) {
	var user User
	if err := client.Get(hostname, "user", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// RESTPrefix returns the REST API base URL for a hostname.
// This is exported for use by other packages.
func RESTPrefix(hostname string) string {
	return bbinstance.RESTPrefix(hostname)
}
