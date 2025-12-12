package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ReviewOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string
	ReviewType  string // "approve" or "unapprove" or "request-changes"
	Body        string
}

func NewCmdReview(f *cmdutil.Factory, runF func(*ReviewOptions) error) *cobra.Command {
	opts := &ReviewOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	var (
		flagApprove        bool
		flagUnapprove      bool
		flagRequestChanges bool
	)

	cmd := &cobra.Command{
		Use:   "review [<number> | <url>]",
		Short: "Add a review to a pull request",
		Long: heredoc.Doc(`
			Add a review to a pull request.

			Without an argument, the pull request that belongs to the current branch is reviewed.

			Note: Bitbucket uses "request changes" to indicate the reviewer has concerns that
			should be addressed before merging.
		`),
		Example: heredoc.Doc(`
			# Approve the pull request
			$ bb pr review 123 --approve

			# Remove your approval
			$ bb pr review 123 --unapprove

			# Request changes on a pull request
			$ bb pr review 123 --request-changes
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			found := 0
			if flagApprove {
				found++
				opts.ReviewType = "approve"
			}
			if flagUnapprove {
				found++
				opts.ReviewType = "unapprove"
			}
			if flagRequestChanges {
				found++
				opts.ReviewType = "request-changes"
			}

			if found == 0 {
				return cmdutil.FlagErrorf("--approve, --unapprove, or --request-changes required")
			}
			if found > 1 {
				return cmdutil.FlagErrorf("need exactly one of --approve, --unapprove, or --request-changes")
			}

			if runF != nil {
				return runF(opts)
			}
			return reviewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&flagApprove, "approve", "a", false, "Approve pull request")
	cmd.Flags().BoolVarP(&flagUnapprove, "unapprove", "u", false, "Remove your approval from pull request")
	cmd.Flags().BoolVarP(&flagRequestChanges, "request-changes", "r", false, "Request changes on a pull request")

	return cmd
}

func reviewRun(opts *ReviewOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the PR argument first to check if it contains repo info
	prID, prRepo, err := shared.ParsePRArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if prRepo != nil {
		repo = prRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	cs := opts.IO.ColorScheme()

	switch opts.ReviewType {
	case "approve":
		err = approvePR(httpClient, repo, prID)
		if err != nil {
			return fmt.Errorf("failed to approve pull request: %w", err)
		}
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "%s Approved pull request #%d\n", cs.SuccessIcon(), prID)
		}

	case "unapprove":
		err = unapprovePR(httpClient, repo, prID)
		if err != nil {
			return fmt.Errorf("failed to unapprove pull request: %w", err)
		}
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "%s Removed approval from pull request #%d\n", cs.Yellow("!"), prID)
		}

	case "request-changes":
		err = requestChangesPR(httpClient, repo, prID)
		if err != nil {
			return fmt.Errorf("failed to request changes: %w", err)
		}
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "%s Requested changes on pull request #%d\n", cs.Red("+"), prID)
		}
	}

	return nil
}

// approvePR approves a pull request
// POST /2.0/repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/approve
func approvePR(client *http.Client, repo bbrepo.Interface, prID int) error {
	url := fmt.Sprintf("%srepositories/%s/%s/pullrequests/%d/approve",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		prID,
	)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("pull request #%d not found", prID)
	}

	// 400 Bad Request often means you can't approve your own PR
	if resp.StatusCode == http.StatusBadRequest {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error.Message != "" {
			return fmt.Errorf("%s", errResp.Error.Message)
		}
		return fmt.Errorf("cannot approve pull request (you may not be able to approve your own PR)")
	}

	// 200 OK or 409 Conflict (already approved) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return api.HandleHTTPError(resp)
	}

	return nil
}

// unapprovePR removes approval from a pull request
// DELETE /2.0/repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/approve
func unapprovePR(client *http.Client, repo bbrepo.Interface, prID int) error {
	url := fmt.Sprintf("%srepositories/%s/%s/pullrequests/%d/approve",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		prID,
	)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("pull request #%d not found or not approved", prID)
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return api.HandleHTTPError(resp)
	}

	return nil
}

// requestChangesPR requests changes on a pull request
// POST /2.0/repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/request-changes
func requestChangesPR(client *http.Client, repo bbrepo.Interface, prID int) error {
	url := fmt.Sprintf("%srepositories/%s/%s/pullrequests/%d/request-changes",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		prID,
	)

	// Create empty JSON body
	body := bytes.NewBufferString("{}")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("pull request #%d not found", prID)
	}

	// Check for successful responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		// Try to get error message from response
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error.Message != "" {
			return fmt.Errorf("%s", errResp.Error.Message)
		}
		return api.HandleHTTPError(resp)
	}

	return nil
}
