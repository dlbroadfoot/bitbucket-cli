package checkout

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/git"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/pkg/cmd/pr/list"
	"github.com/cli/bb/v2/pkg/cmd/pr/shared"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type CheckoutOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	GitClient  *git.Client

	SelectorArg string
	BranchName  string
	Force       bool
}

func NewCmdCheckout(f *cmdutil.Factory, runF func(*CheckoutOptions) error) *cobra.Command {
	opts := &CheckoutOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		GitClient:  f.GitClient,
	}

	cmd := &cobra.Command{
		Use:   "checkout [<number> | <url>]",
		Short: "Check out a pull request in git",
		Long: heredoc.Doc(`
			Check out a pull request in git.

			This command fetches the head branch of the pull request and checks it out locally.
		`),
		Example: heredoc.Doc(`
			# Check out pull request #123
			$ bb pr checkout 123

			# Check out with a custom local branch name
			$ bb pr checkout 123 --branch my-branch
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SelectorArg = args[0]

			if runF != nil {
				return runF(opts)
			}
			return checkoutRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.BranchName, "branch", "b", "", "Local branch name to use")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force checkout even if there are local changes")

	return cmd
}

func checkoutRun(opts *CheckoutOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	// Parse the PR argument
	prID, prRepo, err := shared.ParsePRArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided
	if prRepo != nil {
		repo = prRepo
	}

	// Fetch the PR
	pr, err := list.FetchPullRequest(httpClient, repo, prID)
	if err != nil {
		return err
	}

	// Determine branch name
	branchName := opts.BranchName
	if branchName == "" {
		branchName = pr.HeadBranch()
	}

	cs := opts.IO.ColorScheme()
	ctx := context.Background()

	// Determine remote - use origin if it points to the same repo
	remote := "origin"

	// Fetch the branch
	fmt.Fprintf(opts.IO.ErrOut, "Fetching %s from %s...\n", pr.HeadBranch(), remote)

	// git fetch origin <branch>:<local-branch>
	fetchRefspec := fmt.Sprintf("%s:%s", pr.HeadBranch(), branchName)
	fetchErr := opts.GitClient.Fetch(ctx, remote, fetchRefspec)
	if fetchErr != nil {
		// Try fetching just the branch and creating locally
		fetchErr = opts.GitClient.Fetch(ctx, remote, pr.HeadBranch())
		if fetchErr != nil {
			return fmt.Errorf("failed to fetch branch: %w", fetchErr)
		}

		// Create local branch from fetched ref
		cmd, err := opts.GitClient.Command(ctx, "branch", branchName, fmt.Sprintf("%s/%s", remote, pr.HeadBranch()))
		if err != nil {
			return err
		}
		err = cmd.Run()
		if err != nil {
			// Branch might already exist, try to update it
			cmd, err = opts.GitClient.Command(ctx, "branch", "-f", branchName, fmt.Sprintf("%s/%s", remote, pr.HeadBranch()))
			if err != nil {
				return err
			}
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("failed to create local branch: %w", err)
			}
		}
	}

	// Checkout the branch
	checkoutArgs := []string{"checkout", branchName}
	if opts.Force {
		checkoutArgs = append(checkoutArgs, "--force")
	}

	cmd, err := opts.GitClient.Command(ctx, checkoutArgs...)
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	fmt.Fprintf(opts.IO.Out, "%s Checked out pull request #%d on branch %s\n",
		cs.SuccessIcon(), pr.ID, cs.Cyan(branchName))

	return nil
}
