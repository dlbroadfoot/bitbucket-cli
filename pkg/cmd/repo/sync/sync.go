package sync

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/git"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type SyncOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	GitClient  *git.Client

	Branch string
	Force  bool
}

func NewCmdSync(f *cmdutil.Factory, runF func(*SyncOptions) error) *cobra.Command {
	opts := &SyncOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		GitClient:  f.GitClient,
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync the local repository with the remote",
		Long: heredoc.Doc(`
			Sync the local repository with the remote.

			This command fetches changes from the remote and fast-forwards the
			local branch to match. If the local branch has diverged from the remote,
			use --force to overwrite local changes.

			If no branch is specified, the current branch is synced.
		`),
		Example: heredoc.Doc(`
			$ bb repo sync
			$ bb repo sync --branch main
			$ bb repo sync --force
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return syncRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Branch to sync (defaults to current branch)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force sync even if branches have diverged")

	return cmd
}

func syncRun(opts *SyncOptions) error {
	ctx := context.Background()

	// Get the current branch if not specified
	branch := opts.Branch
	if branch == "" {
		currentBranch, err := opts.GitClient.CurrentBranch(ctx)
		if err != nil {
			return fmt.Errorf("could not determine current branch: %w", err)
		}
		branch = currentBranch
	}

	cs := opts.IO.ColorScheme()

	// Fetch from remote
	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(opts.IO.Out, "Fetching changes from remote...\n")
	}

	opts.IO.StartProgressIndicator()

	// Get the remote name (usually 'origin')
	remote := "origin"

	// Fetch the branch from remote
	fetchCmd, err := opts.GitClient.Command(ctx, "fetch", remote, branch)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to create fetch command: %w", err)
	}

	_, fetchErr := fetchCmd.Output()
	if fetchErr != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to fetch from remote: %w", fetchErr)
	}

	// Check if we can fast-forward
	remoteBranch := fmt.Sprintf("%s/%s", remote, branch)
	mergeBaseCmd, err := opts.GitClient.Command(ctx, "merge-base", branch, remoteBranch)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to create merge-base command: %w", err)
	}

	mergeBase, _ := mergeBaseCmd.Output()

	localRevCmd, err := opts.GitClient.Command(ctx, "rev-parse", branch)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to create rev-parse command: %w", err)
	}
	localRev, _ := localRevCmd.Output()

	remoteRevCmd, err := opts.GitClient.Command(ctx, "rev-parse", remoteBranch)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to create rev-parse command: %w", err)
	}
	remoteRev, _ := remoteRevCmd.Output()

	opts.IO.StopProgressIndicator()

	// Check if already up to date
	if string(localRev) == string(remoteRev) {
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.Out, "%s Branch %s is already up to date\n",
				cs.SuccessIcon(), cs.Bold(branch))
		}
		return nil
	}

	// Check if we can fast-forward
	canFastForward := string(mergeBase) == string(localRev)

	if !canFastForward && !opts.Force {
		return fmt.Errorf("local branch %s has diverged from %s; use --force to overwrite local changes",
			branch, remoteBranch)
	}

	// Perform the sync
	opts.IO.StartProgressIndicator()

	var resetArgs []string
	if opts.Force {
		resetArgs = []string{"reset", "--hard", remoteBranch}
	} else {
		resetArgs = []string{"merge", "--ff-only", remoteBranch}
	}

	resetCmd, err := opts.GitClient.Command(ctx, resetArgs...)
	if err != nil {
		opts.IO.StopProgressIndicator()
		return fmt.Errorf("failed to create sync command: %w", err)
	}

	_, resetErr := resetCmd.Output()
	opts.IO.StopProgressIndicator()

	if resetErr != nil {
		return fmt.Errorf("failed to sync branch: %w", resetErr)
	}

	if opts.IO.IsStdoutTTY() {
		if opts.Force {
			fmt.Fprintf(opts.IO.Out, "%s Force synced branch %s with %s\n",
				cs.SuccessIcon(), cs.Bold(branch), cs.Cyan(remoteBranch))
		} else {
			fmt.Fprintf(opts.IO.Out, "%s Synced branch %s with %s\n",
				cs.SuccessIcon(), cs.Bold(branch), cs.Cyan(remoteBranch))
		}
	}

	return nil
}
