package edit

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/list"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/issue/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type EditOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)

	SelectorArg string

	Title    string
	Body     string
	BodyFile string
	State    string
	Priority string
	Kind     string
	Assignee string
}

func NewCmdEdit(f *cmdutil.Factory, runF func(*EditOptions) error) *cobra.Command {
	opts := &EditOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit an issue",
		Long: heredoc.Doc(`
			Edit an issue.

			You can modify the title, description, state, priority, kind, or assignee
			of an issue.
		`),
		Example: heredoc.Doc(`
			$ bb issue edit 123 --title "New title"
			$ bb issue edit 123 --body "New description"
			$ bb issue edit 123 --state open
			$ bb issue edit 123 --priority major
			$ bb issue edit 123 --kind bug
			$ bb issue edit 123 --assignee username
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if opts.BodyFile != "" {
				b, err := cmdutil.ReadFile(opts.BodyFile, opts.IO.In)
				if err != nil {
					return err
				}
				opts.Body = string(b)
			}

			// Validate that at least one edit flag is provided
			hasEdits := opts.Title != "" ||
				opts.Body != "" ||
				opts.State != "" ||
				opts.Priority != "" ||
				opts.Kind != "" ||
				opts.Assignee != ""

			if !hasEdits {
				return cmdutil.FlagErrorf("at least one edit flag is required: --title, --body, --body-file, --state, --priority, --kind, or --assignee")
			}

			// Validate state if provided
			if opts.State != "" {
				validStates := []string{"new", "open", "resolved", "on hold", "invalid", "duplicate", "wontfix", "closed"}
				valid := false
				for _, s := range validStates {
					if opts.State == s {
						valid = true
						break
					}
				}
				if !valid {
					return cmdutil.FlagErrorf("invalid state %q, valid states are: new, open, resolved, on hold, invalid, duplicate, wontfix, closed", opts.State)
				}
			}

			// Validate priority if provided
			if opts.Priority != "" {
				validPriorities := []string{"trivial", "minor", "major", "critical", "blocker"}
				valid := false
				for _, p := range validPriorities {
					if opts.Priority == p {
						valid = true
						break
					}
				}
				if !valid {
					return cmdutil.FlagErrorf("invalid priority %q, valid priorities are: trivial, minor, major, critical, blocker", opts.Priority)
				}
			}

			// Validate kind if provided
			if opts.Kind != "" {
				validKinds := []string{"bug", "enhancement", "proposal", "task"}
				valid := false
				for _, k := range validKinds {
					if opts.Kind == k {
						valid = true
						break
					}
				}
				if !valid {
					return cmdutil.FlagErrorf("invalid kind %q, valid kinds are: bug, enhancement, proposal, task", opts.Kind)
				}
			}

			if runF != nil {
				return runF(opts)
			}
			return editRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Set the new title")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Set the new description")
	cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read description from `file` (use \"-\" for stdin)")
	cmd.Flags().StringVarP(&opts.State, "state", "s", "", "Set the state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "", "Set the priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "", "Set the kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Set the assignee by username")

	return cmd
}

func editRun(opts *EditOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the issue argument
	issueID, issueRepo, err := shared.ParseIssueArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if issueRepo != nil {
		repo = issueRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	// Fetch the current issue to verify it exists
	issue, err := list.FetchIssue(httpClient, repo, issueID)
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()

	// Build the update payload
	update := buildUpdatePayload(opts)

	// Update the issue
	err = updateIssue(httpClient, repo, issueID, update)

	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Edited issue #%d\n", cs.SuccessIcon(), issueID)
		fmt.Fprintln(opts.IO.Out, issue.HTMLURL())
	}

	return nil
}

// IssueUpdatePayload represents the fields that can be updated on an issue
type IssueUpdatePayload struct {
	Title    string         `json:"title,omitempty"`
	Content  *ContentUpdate `json:"content,omitempty"`
	State    string         `json:"state,omitempty"`
	Priority string         `json:"priority,omitempty"`
	Kind     string         `json:"kind,omitempty"`
	Assignee *UserRef       `json:"assignee,omitempty"`
}

type ContentUpdate struct {
	Raw string `json:"raw"`
}

type UserRef struct {
	Username string `json:"username,omitempty"`
}

func buildUpdatePayload(opts *EditOptions) IssueUpdatePayload {
	update := IssueUpdatePayload{}

	if opts.Title != "" {
		update.Title = opts.Title
	}

	if opts.Body != "" {
		update.Content = &ContentUpdate{Raw: opts.Body}
	}

	if opts.State != "" {
		update.State = opts.State
	}

	if opts.Priority != "" {
		update.Priority = opts.Priority
	}

	if opts.Kind != "" {
		update.Kind = opts.Kind
	}

	if opts.Assignee != "" {
		update.Assignee = &UserRef{Username: opts.Assignee}
	}

	return update
}

func updateIssue(client *http.Client, repo bbrepo.Interface, issueID int, payload IssueUpdatePayload) error {
	apiClient := api.NewClientFromHTTP(client)

	path := fmt.Sprintf("repositories/%s/%s/issues/%d",
		repo.RepoWorkspace(), repo.RepoSlug(), issueID)

	return apiClient.Put(repo.RepoHost(), path, payload, nil)
}
