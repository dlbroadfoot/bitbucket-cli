# Bitbucket CLI (`bb`)

Bitbucket on the command line. Pull requests, issues, pipelines, and more — right next to where you work with `git`.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/bb/script/install.sh | sh
```

Or install from source (Go 1.21+):

```bash
go install github.com/dlbroadfoot/bitbucket-cli/bb/cmd/bb@latest
```

Pre-built binaries are also available on the [releases page](https://github.com/dlbroadfoot/bitbucket-cli/releases).

## Authentication

```bash
bb auth login
```

You'll need a Bitbucket API token. Create one at:
https://id.atlassian.com/manage-profile/security/api-tokens

**Required scopes:** `read:user`, `read:account`, `read:repository`, `write:repository`, `read:pullrequest`, `write:pullrequest`

Alternatively, set the `BB_TOKEN` environment variable (`email:api_token` format).

## Quick Start

```bash
bb repo list --workspace myworkspace    # List repositories
bb repo clone myworkspace/myrepo        # Clone a repository
bb pr list                              # List pull requests
bb pr create --title "My PR"            # Create a pull request
bb issue list                           # List issues
bb pipeline list                        # View pipeline runs
bb browse                               # Open repo in browser
```

## Commands

| Command | Description |
|---------|-------------|
| `bb auth` | Authenticate with Bitbucket |
| `bb repo` | Manage repositories (list, clone, create, view, edit, delete, sync, fork) |
| `bb pr` | Manage pull requests (list, view, create, checkout, merge, close, diff, checks) |
| `bb issue` | Manage issues (list, view, create, edit, close, reopen, comment) |
| `bb pipeline` | Manage pipelines (list, view, run, cancel) |
| `bb project` | Work with Bitbucket projects (list, view, create) |
| `bb workspace` | Manage workspaces (list, view) |
| `bb search` | Search repositories and code |
| `bb ssh-key` | Manage SSH keys |
| `bb secret` | Manage repository secrets |
| `bb variable` | Manage pipeline variables |
| `bb browse` | Open repository in browser |
| `bb api` | Make authenticated API requests |
| `bb status` | View status across workspaces |
| `bb config` | Manage configuration |
| `bb alias` | Create command shortcuts |

## Shell Completion

```bash
bb completion -s bash > /etc/bash_completion.d/bb   # Bash
bb completion -s zsh > "${fpath[1]}/_bb"             # Zsh
bb completion -s fish > ~/.config/fish/completions/bb.fish  # Fish
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BB_TOKEN` | Authentication token (`email:api_token` format) |
| `BB_HOST` | Bitbucket host (default: `bitbucket.org`) |
| `BB_REPO` | Default repository (`WORKSPACE/REPO` format) |
| `BB_PAGER` | Pager program (default: `less`) |
| `NO_COLOR` | Disable colored output |

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Based on the [GitHub CLI](https://github.com/cli/cli) architecture.
