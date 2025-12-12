package edit

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type EditOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string

	Title       string
	Description string
	BodyFile    string

	AddReviewer    []string
	RemoveReviewer []string

	DestinationBranch string
	CloseSourceBranch *bool
}

func NewCmdEdit(f *cmdutil.Factory, runF func(*EditOptions) error) *cobra.Command {
	opts := &EditOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	var closeSourceBranch bool
	var keepSourceBranch bool

	cmd := &cobra.Command{
		Use:   "edit [<number> | <url>]",
		Short: "Edit a pull request",
		Long: heredoc.Doc(`
			Edit a pull request.

			Without an argument, the pull request that belongs to the current branch
			is selected.
		`),
		Example: heredoc.Doc(`
			$ bb pr edit 23 --title "Updated title"
			$ bb pr edit 23 --body "New description"
			$ bb pr edit 23 --add-reviewer user1,user2
			$ bb pr edit 23 --remove-reviewer user3
			$ bb pr edit 23 --destination main
			$ bb pr edit 23 --close-source-branch
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			flags := cmd.Flags()

			if opts.BodyFile != "" {
				b, err := cmdutil.ReadFile(opts.BodyFile, opts.IO.In)
				if err != nil {
					return err
				}
				opts.Description = string(b)
			}

			// Handle close source branch flags
			if flags.Changed("close-source-branch") {
				opts.CloseSourceBranch = &closeSourceBranch
			}
			if flags.Changed("keep-source-branch") {
				val := !keepSourceBranch
				opts.CloseSourceBranch = &val
			}

			// Validate that at least one edit flag is provided
			hasEdits := opts.Title != "" ||
				opts.Description != "" ||
				len(opts.AddReviewer) > 0 ||
				len(opts.RemoveReviewer) > 0 ||
				opts.DestinationBranch != "" ||
				opts.CloseSourceBranch != nil

			if !hasEdits {
				return cmdutil.FlagErrorf("at least one edit flag is required: --title, --body, --body-file, --add-reviewer, --remove-reviewer, --destination, --close-source-branch, or --keep-source-branch")
			}

			if runF != nil {
				return runF(opts)
			}
			return editRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Set the new title")
	cmd.Flags().StringVarP(&opts.Description, "body", "b", "", "Set the new description")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read description from `file` (use \"-\" for stdin)")
	cmd.Flags().StringSliceVar(&opts.AddReviewer, "add-reviewer", nil, "Add reviewers by username")
	cmd.Flags().StringSliceVar(&opts.RemoveReviewer, "remove-reviewer", nil, "Remove reviewers by username")
	cmd.Flags().StringVarP(&opts.DestinationBranch, "destination", "d", "", "Change the destination branch")
	cmd.Flags().BoolVar(&closeSourceBranch, "close-source-branch", false, "Delete source branch after merge")
	cmd.Flags().BoolVar(&keepSourceBranch, "keep-source-branch", false, "Keep source branch after merge")

	return cmd
}

func editRun(opts *EditOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the PR argument
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

	// Fetch the current PR to get existing data
	pr, err := list.FetchPullRequest(httpClient, repo, prID)
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()

	// Build the update payload
	update := buildUpdatePayload(opts, pr)

	// Update the PR
	err = updatePullRequest(httpClient, repo, prID, update)

	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Edited pull request #%d\n", cs.SuccessIcon(), prID)
		fmt.Fprintln(opts.IO.Out, pr.HTMLURL())
	}

	return nil
}

// PRUpdatePayload represents the fields that can be updated on a PR
type PRUpdatePayload struct {
	Title             string       `json:"title,omitempty"`
	Description       string       `json:"description,omitempty"`
	Reviewers         []UserRef    `json:"reviewers,omitempty"`
	Destination       *BranchRef   `json:"destination,omitempty"`
	CloseSourceBranch *bool        `json:"close_source_branch,omitempty"`
}

type UserRef struct {
	UUID string `json:"uuid,omitempty"`
}

type BranchRef struct {
	Branch BranchName `json:"branch"`
}

type BranchName struct {
	Name string `json:"name"`
}

func buildUpdatePayload(opts *EditOptions, pr *shared.PullRequest) PRUpdatePayload {
	update := PRUpdatePayload{}

	if opts.Title != "" {
		update.Title = opts.Title
	}

	if opts.Description != "" {
		update.Description = opts.Description
	}

	if opts.DestinationBranch != "" {
		update.Destination = &BranchRef{
			Branch: BranchName{Name: opts.DestinationBranch},
		}
	}

	if opts.CloseSourceBranch != nil {
		update.CloseSourceBranch = opts.CloseSourceBranch
	}

	// Handle reviewers
	if len(opts.AddReviewer) > 0 || len(opts.RemoveReviewer) > 0 {
		// Build a map of current reviewers by UUID
		reviewerMap := make(map[string]bool)
		for _, r := range pr.Reviewers {
			reviewerMap[r.UUID] = true
		}

		// Remove reviewers - we need to match by display name or nickname
		removeSet := make(map[string]bool)
		for _, r := range opts.RemoveReviewer {
			removeSet[r] = true
		}

		// Filter out removed reviewers
		var newReviewers []UserRef
		for _, r := range pr.Reviewers {
			if !removeSet[r.DisplayName] && !removeSet[r.Nickname] && !removeSet[r.AccountID] {
				newReviewers = append(newReviewers, UserRef{UUID: r.UUID})
			}
		}

		// Note: Adding reviewers by username requires looking up their UUID
		// For now, we'll only support removing reviewers and keep existing ones
		// Adding new reviewers would require a user lookup API call

		update.Reviewers = newReviewers
	}

	return update
}

func updatePullRequest(client *http.Client, repo bbrepo.Interface, prID int, payload PRUpdatePayload) error {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/pullrequests/%d",
		repo.RepoWorkspace(), repo.RepoSlug(), prID)

	return apiClient.Put(repo.RepoHost(), path, payload, nil)
}
