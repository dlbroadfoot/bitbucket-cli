package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/internal/bbinstance"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/internal/gh"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ApiOptions struct {
	IO         *iostreams.IOStreams
	HttpClient func() (*http.Client, error)
	Config     func() (gh.Config, error)
	BaseRepo   func() (bbrepo.Interface, error)

	Hostname        string
	RequestMethod   string
	RequestPath     string
	RequestBody     string
	RequestBodyFile string
	RawFields       []string
	MagicFields     []string
	RequestHeaders  []string
	Silent          bool
	Paginate        bool
	JQ              string
}

func NewCmdApi(f *cmdutil.Factory, runF func(*ApiOptions) error) *cobra.Command {
	opts := &ApiOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Config:     f.Config,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated API request",
		Long: heredoc.Docf(`
			Makes an authenticated HTTP request to the Bitbucket API and prints the response.

			The endpoint argument should be a path of a Bitbucket API v2.0 endpoint.

			The default HTTP request method is "GET" normally and "POST" if any parameters
			were added. Override the method with %[1]s--method%[1]s.

			Pass one or more %[1]s-f/--raw-field%[1]s values in "key=value" format to add JSON-encoded
			parameters to the POST body.

			Pass one or more %[1]s-F/--field%[1]s values in "key=value" format to add JSON parameters
			to the POST body. The value will be parsed as JSON if it looks like a number, boolean,
			array, or object.

			Add a HTTP request header with %[1]s-H/--header%[1]s in "key:value" format.

			The endpoint path can include placeholders like "{workspace}" and "{repo_slug}" which
			will be replaced with values from the current repository context.
		`, "`"),
		Example: heredoc.Doc(`
			# List pull requests for the current repository
			$ bb api repositories/{workspace}/{repo_slug}/pullrequests

			# Create a pull request
			$ bb api repositories/{workspace}/{repo_slug}/pullrequests -f title="My PR" -f source='{"branch":{"name":"feature"}}'

			# Get current user
			$ bb api user

			# Get workspaces
			$ bb api workspaces

			# List repositories in a workspace
			$ bb api repositories/myworkspace

			# Use with jq for filtering
			$ bb api repositories/{workspace}/{repo_slug}/pullrequests | jq '.values[].title'
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RequestPath = args[0]

			if runF != nil {
				return runF(opts)
			}
			return apiRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", "The Bitbucket hostname for the request (default \"bitbucket.org\")")
	cmd.Flags().StringVarP(&opts.RequestMethod, "method", "X", "", "The HTTP method for the request")
	cmd.Flags().StringArrayVarP(&opts.MagicFields, "field", "F", nil, "Add a parameter with JSON value")
	cmd.Flags().StringArrayVarP(&opts.RawFields, "raw-field", "f", nil, "Add a string parameter")
	cmd.Flags().StringArrayVarP(&opts.RequestHeaders, "header", "H", nil, "Add a HTTP request header")
	cmd.Flags().StringVar(&opts.RequestBody, "input", "", "The file to use as body for the HTTP request (use \"-\" to read from stdin)")
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, "Do not print the response body")
	cmd.Flags().BoolVar(&opts.Paginate, "paginate", false, "Make additional HTTP requests to fetch all pages of results")
	cmd.Flags().StringVarP(&opts.JQ, "jq", "q", "", "Query to select values from the response using jq syntax")

	return cmd
}

func apiRun(opts *ApiOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	hostname := opts.Hostname
	if hostname == "" {
		cfg, err := opts.Config()
		if err == nil {
			hostname, _ = cfg.Authentication().DefaultHost()
		}
		if hostname == "" {
			hostname = bbinstance.Default()
		}
	}

	// Replace placeholders in path
	requestPath := opts.RequestPath
	if strings.Contains(requestPath, "{workspace}") || strings.Contains(requestPath, "{repo_slug}") {
		repo, err := opts.BaseRepo()
		if err == nil {
			requestPath = strings.ReplaceAll(requestPath, "{workspace}", repo.RepoWorkspace())
			requestPath = strings.ReplaceAll(requestPath, "{repo_slug}", repo.RepoSlug())
		}
	}

	// Build the full URL
	var requestURL string
	if strings.HasPrefix(requestPath, "http://") || strings.HasPrefix(requestPath, "https://") {
		requestURL = requestPath
	} else {
		requestURL = bbinstance.RESTPrefix(hostname) + strings.TrimPrefix(requestPath, "/")
	}

	// Determine method
	method := opts.RequestMethod
	if method == "" {
		if len(opts.RawFields) > 0 || len(opts.MagicFields) > 0 || opts.RequestBody != "" {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	// Build request body
	var body io.Reader
	if opts.RequestBody != "" {
		if opts.RequestBody == "-" {
			body = opts.IO.In
		} else {
			f, err := os.Open(opts.RequestBody)
			if err != nil {
				return fmt.Errorf("failed to open input file: %w", err)
			}
			defer f.Close()
			body = f
		}
	} else if len(opts.RawFields) > 0 || len(opts.MagicFields) > 0 {
		params := make(map[string]interface{})

		for _, field := range opts.RawFields {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s", field)
			}
			params[parts[0]] = parts[1]
		}

		for _, field := range opts.MagicFields {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s", field)
			}
			var value interface{}
			if err := json.Unmarshal([]byte(parts[1]), &value); err != nil {
				params[parts[0]] = parts[1] // treat as string if not valid JSON
			} else {
				params[parts[0]] = value
			}
		}

		jsonBody, err := json.Marshal(params)
		if err != nil {
			return err
		}
		body = bytes.NewReader(jsonBody)
	}

	for {
		req, err := http.NewRequest(method, requestURL, body)
		if err != nil {
			return err
		}

		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		for _, header := range opts.RequestHeaders {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid header format: %s", header)
			}
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode >= 400 {
			defer resp.Body.Close()
			return api.HandleHTTPError(resp)
		}

		if !opts.Silent {
			// Read and print response
			responseBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}

			// Pretty print JSON
			var prettyJSON bytes.Buffer
			if json.Indent(&prettyJSON, responseBody, "", "  ") == nil {
				fmt.Fprintln(opts.IO.Out, prettyJSON.String())
			} else {
				fmt.Fprintln(opts.IO.Out, string(responseBody))
			}

			// Check for pagination
			if opts.Paginate {
				var result map[string]interface{}
				if err := json.Unmarshal(responseBody, &result); err == nil {
					if next, ok := result["next"].(string); ok && next != "" {
						requestURL = next
						body = nil // Don't resend body on pagination
						continue
					}
				}
			}
		} else {
			resp.Body.Close()
		}

		break
	}

	return nil
}

// ParseURL parses a URL and returns the host and path components
func ParseURL(urlStr string) (*url.URL, error) {
	return url.Parse(urlStr)
}
