package api

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbinstance"
	"github.com/dlbroadfoot/bitbucket-cli/utils"
)

type tokenGetter interface {
	ActiveToken(string) (string, string)
}

// HTTPClientOptions configures the HTTP client.
type HTTPClientOptions struct {
	AppVersion         string
	CacheTTL           time.Duration
	Config             tokenGetter
	EnableCache        bool
	Log                io.Writer
	LogColorize        bool
	LogVerboseHTTP     bool
	SkipDefaultHeaders bool
}

// NewHTTPClient creates a new HTTP client configured for the Bitbucket API.
func NewHTTPClient(opts HTTPClientOptions) (*http.Client, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Build transport chain
	var transport http.RoundTripper = http.DefaultTransport

	// Add default headers
	if !opts.SkipDefaultHeaders {
		transport = &headerTripper{
			base: transport,
			headers: map[string]string{
				userAgent: fmt.Sprintf("Bitbucket CLI %s", opts.AppVersion),
				accept:    "application/json",
			},
		}
	}

	// Add logging if debug is enabled
	debugEnabled, debugValue := utils.IsDebugEnabled()
	if strings.Contains(debugValue, "api") {
		opts.LogVerboseHTTP = true
	}

	if (opts.LogVerboseHTTP || debugEnabled) && opts.Log != nil {
		transport = &loggingTripper{
			base:     transport,
			log:      opts.Log,
			colorize: opts.LogColorize,
			verbose:  opts.LogVerboseHTTP,
		}
	}

	// Add authentication
	if opts.Config != nil {
		transport = AddBasicAuthHeader(transport, opts.Config)
	}

	client.Transport = transport

	return client, nil
}

// AddBasicAuthHeader adds Basic Auth header for Bitbucket API requests.
// Bitbucket uses Basic Auth with email:api_token format.
func AddBasicAuthHeader(rt http.RoundTripper, cfg tokenGetter) http.RoundTripper {
	return &funcTripper{roundTrip: func(req *http.Request) (*http.Response, error) {
		// If the header is already set in the request, don't overwrite it.
		if req.Header.Get(authorization) == "" {
			var redirectHostnameChange bool
			if req.Response != nil && req.Response.Request != nil {
				redirectHostnameChange = getHost(req) != getHost(req.Response.Request)
			}
			// Only set header if an initial request or redirect request to the same host as the initial request.
			// If the host has changed during a redirect do not add the authentication token header.
			if !redirectHostnameChange {
				hostname := bbinstance.NormalizeHostname(getHost(req))
				if token, _ := cfg.ActiveToken(hostname); token != "" {
					// Bitbucket tokens are stored as "email:api_token"
					// Use Basic Auth encoding
					encoded := base64.StdEncoding.EncodeToString([]byte(token))
					req.Header.Set(authorization, fmt.Sprintf("Basic %s", encoded))
				}
			}
		}
		return rt.RoundTrip(req)
	}}
}

// headerTripper adds default headers to requests.
type headerTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range t.headers {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}
	return t.base.RoundTrip(req)
}

// loggingTripper logs HTTP requests and responses.
type loggingTripper struct {
	base     http.RoundTripper
	log      io.Writer
	colorize bool
	verbose  bool
}

func (t *loggingTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request
	fmt.Fprintf(t.log, "* Request: %s %s\n", req.Method, req.URL)
	if t.verbose {
		for key, values := range req.Header {
			// Don't log sensitive headers
			if strings.EqualFold(key, "Authorization") {
				fmt.Fprintf(t.log, "> %s: [REDACTED]\n", key)
			} else {
				for _, value := range values {
					fmt.Fprintf(t.log, "> %s: %s\n", key, value)
				}
			}
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(t.log, "* Error: %v\n", err)
		return resp, err
	}

	// Log response
	fmt.Fprintf(t.log, "* Response: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	if t.verbose {
		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(t.log, "< %s: %s\n", key, value)
			}
		}
	}

	return resp, nil
}

// funcTripper wraps a function as an http.RoundTripper.
type funcTripper struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (tr funcTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return tr.roundTrip(req)
}

func getHost(r *http.Request) string {
	if r.Host != "" {
		return r.Host
	}
	return r.URL.Host
}

// ExtractHeader extracts a named header from any response received by this client and,
// if non-blank, saves it to dest.
func ExtractHeader(name string, dest *string) func(http.RoundTripper) http.RoundTripper {
	return func(tr http.RoundTripper) http.RoundTripper {
		return &funcTripper{roundTrip: func(req *http.Request) (*http.Response, error) {
			res, err := tr.RoundTrip(req)
			if err == nil {
				if value := res.Header.Get(name); value != "" {
					*dest = value
				}
			}
			return res, err
		}}
	}
}
