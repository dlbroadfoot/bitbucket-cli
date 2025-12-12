package clone

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/bb/v2/api"
	"github.com/cli/bb/v2/git"
	"github.com/cli/bb/v2/internal/bbrepo"
	"github.com/cli/bb/v2/internal/gh"
	"github.com/cli/bb/v2/pkg/cmdutil"
	"github.com/cli/bb/v2/pkg/iostreams"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CloneOptions struct {
	HttpClient func() (*http.Client, error)
	GitClient  *git.Client
	Config     func() (gh.Config, error)
	IO         *iostreams.IOStreams

	GitArgs      []string
	Repository   string
	UpstreamName string
}

func NewCmdClone(f *cmdutil.Factory, runF func(*CloneOptions) error) *cobra.Command {
	opts := &CloneOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		GitClient:  f.GitClient,
		Config:     f.Config,
	}

	cmd := &cobra.Command{
		DisableFlagsInUseLine: true,

		Use:   "clone <repository> [<directory>] [-- <gitflags>...]",
		Args:  cmdutil.MinimumArgs(1, "cannot clone: repository argument required"),
		Short: "Clone a repository locally",
		Long: heredoc.Docf(`
			Clone a Bitbucket repository locally. Pass additional %[1]sgit clone%[1]s flags by listing
			them after %[1]s--%[1]s.

			If the %[1]sWORKSPACE/%[1]s portion of the %[1]sWORKSPACE/REPO%[1]s repository argument is omitted, it
			defaults to the name of the authenticating user.

			When a protocol scheme is not provided in the repository argument, the %[1]sgit_protocol%[1]s will be
			chosen from your configuration, which can be checked via %[1]sbb config get git_protocol%[1]s. If the protocol
			scheme is provided, the repository will be cloned using the specified protocol.

			If the repository is a fork, its parent repository will be added as an additional
			git remote called %[1]supstream%[1]s. The remote name can be configured using %[1]s--upstream-remote-name%[1]s.
		`, "`"),
		Example: heredoc.Doc(`
			# Clone a repository from a specific workspace
			$ bb repo clone atlassian/bitbucket-cli

			# Clone a repository from your own workspace
			$ bb repo clone myrepo

			# Clone a repo using URL
			$ bb repo clone https://bitbucket.org/atlassian/bitbucket-cli
			$ bb repo clone git@bitbucket.org:atlassian/bitbucket-cli.git

			# Clone a repository to a custom directory
			$ bb repo clone atlassian/bitbucket-cli workspace/cli

			# Clone a repository with additional git clone flags
			$ bb repo clone atlassian/bitbucket-cli -- --depth=1
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Repository = args[0]
			opts.GitArgs = args[1:]

			if runF != nil {
				return runF(opts)
			}

			return cloneRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.UpstreamName, "upstream-remote-name", "u", "upstream", "Upstream remote name when cloning a fork")
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == pflag.ErrHelp {
			return err
		}
		return cmdutil.FlagErrorf("%w\nSeparate git clone flags with '--'.", err)
	})

	return cmd
}

func cloneRun(opts *CloneOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	apiClient := api.NewClientFromHTTP(httpClient)

	repositoryIsURL := strings.Contains(opts.Repository, ":")
	repositoryIsFullName := !repositoryIsURL && strings.Contains(opts.Repository, "/")

	var repo bbrepo.Interface
	var protocol string

	if repositoryIsURL {
		initialURL, err := git.ParseURL(opts.Repository)
		if err != nil {
			return err
		}

		repoURL := simplifyURL(initialURL)
		repo, err = bbrepo.FromURL(repoURL)
		if err != nil {
			return err
		}
		if repoURL.Scheme == "git+ssh" {
			repoURL.Scheme = "ssh"
		}
		protocol = repoURL.Scheme
	} else {
		var fullName string
		if repositoryIsFullName {
			fullName = opts.Repository
		} else {
			host, _ := cfg.Authentication().DefaultHost()
			currentUser, err := api.CurrentLoginName(apiClient, host)
			if err != nil {
				return err
			}
			fullName = currentUser + "/" + opts.Repository
		}

		repo, err = bbrepo.FromFullName(fullName)
		if err != nil {
			return err
		}

		protocol = cfg.GitProtocol(repo.RepoHost()).Value
	}

	// For Bitbucket, we can clone directly without fetching repo info first
	// since we have the workspace/repo_slug already
	cloneURL := bbrepo.FormatRemoteURL(repo, protocol)

	gitClient := opts.GitClient
	ctx := context.Background()
	cloneDir, err := gitClient.Clone(ctx, cloneURL, opts.GitArgs)
	if err != nil {
		return err
	}

	// Try to fetch repo info to check if it's a fork
	var repoInfo api.Repository
	repoPath := fmt.Sprintf("repositories/%s/%s", repo.RepoWorkspace(), repo.RepoSlug())
	if err := apiClient.Get(repo.RepoHost(), repoPath, &repoInfo); err == nil && repoInfo.Parent != nil {
		// It's a fork, add parent as upstream
		parentWorkspace := repoInfo.Parent.Workspace.Slug
		parentSlug := repoInfo.Parent.Slug
		parentRepo := bbrepo.NewWithHost(parentWorkspace, parentSlug, repo.RepoHost())

		upstreamURL := bbrepo.FormatRemoteURL(parentRepo, protocol)

		upstreamName := opts.UpstreamName
		if opts.UpstreamName == "@owner" {
			upstreamName = parentWorkspace
		}

		gc := gitClient.Copy()
		gc.RepoDir = cloneDir

		mainBranch := "main"
		if repoInfo.Parent.MainBranch != nil {
			mainBranch = repoInfo.Parent.MainBranch.Name
		}

		if _, err := gc.AddRemote(ctx, upstreamName, upstreamURL, []string{mainBranch}); err != nil {
			return err
		}

		if err := gc.Fetch(ctx, upstreamName, ""); err != nil {
			return err
		}

		if err := gc.SetRemoteBranches(ctx, upstreamName, `*`); err != nil {
			return err
		}

		if err = gc.SetRemoteResolution(ctx, upstreamName, "base"); err != nil {
			return err
		}

		connectedToTerminal := opts.IO.IsStdoutTTY()
		if connectedToTerminal {
			cs := opts.IO.ColorScheme()
			fmt.Fprintf(opts.IO.ErrOut, "%s Repository %s set as the default repository.\n", cs.WarningIcon(), cs.Bold(bbrepo.FullName(parentRepo)))
		}
	}
	return nil
}

// simplifyURL strips given URL of extra parts like extra path segments (i.e.,
// anything beyond `/workspace/repo`), query strings, or fragments.
func simplifyURL(u *url.URL) *url.URL {
	result := &url.URL{
		Scheme: u.Scheme,
		User:   u.User,
		Host:   u.Host,
		Path:   u.Path,
	}

	pathParts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(pathParts) <= 2 {
		return result
	}

	result.Path = strings.Join(pathParts[0:2], "/")
	return result
}
