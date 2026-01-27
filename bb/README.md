# Bitbucket CLI

`bb` is Bitbucket on the command line. It brings pull requests, issues, pipelines, and other Bitbucket concepts to the terminal next to where you are already working with `git` and your code.

## Features

- **Pull Requests**: Create, view, list, checkout, merge, and close pull requests
- **Issues**: Create, view, list, edit, and manage issues
- **Repositories**: Clone, create, fork, view, and manage repositories
- **Pipelines**: View, run, and cancel pipeline builds
- **Projects**: List and view Bitbucket projects
- **Workspaces**: List and view workspaces
- **Search**: Search for repositories and code
- **SSH Keys**: Manage SSH keys for your account
- **Secrets & Variables**: Manage pipeline secrets and variables

## Installation

### From Source (Go 1.21+)

```bash
go install github.com/dlbroadfoot/bitbucket-cli/bb/cmd/bb@latest
```

This installs the `bb` binary to `$GOPATH/bin` (typically `~/go/bin`).

### From Releases

Download pre-built binaries from the [releases page](https://github.com/dlbroadfoot/bitbucket-cli/releases).

#### macOS

```bash
# Download and extract (Intel)
curl -LO https://github.com/dlbroadfoot/bitbucket-cli/releases/latest/download/bb_VERSION_macOS_x86_64.zip
unzip bb_*_macOS_x86_64.zip
sudo mv bb_*/bb /usr/local/bin/

# Download and extract (Apple Silicon)
curl -LO https://github.com/dlbroadfoot/bitbucket-cli/releases/latest/download/bb_VERSION_macOS_arm64.zip
unzip bb_*_macOS_arm64.zip
sudo mv bb_*/bb /usr/local/bin/
```

#### Linux

```bash
# Download and extract (x86_64)
curl -LO https://github.com/dlbroadfoot/bitbucket-cli/releases/latest/download/bb_VERSION_linux_x86_64.tar.gz
tar xzf bb_*_linux_x86_64.tar.gz
sudo mv bb_*/bb /usr/local/bin/

# Or install via .deb package
curl -LO https://github.com/dlbroadfoot/bitbucket-cli/releases/latest/download/bb_VERSION_linux_amd64.deb
sudo dpkg -i bb_*_linux_amd64.deb

# Or install via .rpm package
curl -LO https://github.com/dlbroadfoot/bitbucket-cli/releases/latest/download/bb_VERSION_linux_amd64.rpm
sudo rpm -i bb_*_linux_amd64.rpm
```

#### Windows

Download the `.zip` file from the [releases page](https://github.com/dlbroadfoot/bitbucket-cli/releases) and add the extracted directory to your PATH.

## Authentication

Before using `bb`, authenticate with Bitbucket:

```bash
bb auth login
```

This will guide you through creating an App Password with the necessary permissions.

**Required App Password Permissions:**
- Account: Read
- Repositories: Read, Write
- Pull Requests: Read, Write
- Issues: Read, Write (if using issue tracker)
- Pipelines: Read, Write (if using pipelines)

## Quick Start

```bash
# Authenticate with Bitbucket
bb auth login

# List repositories in a workspace
bb repo list --workspace myworkspace

# Clone a repository
bb repo clone myworkspace/myrepo

# View the current repository in browser
bb browse

# List pull requests
bb pr list

# Create a pull request
bb pr create --title "My PR" --body "Description"

# View pipeline runs
bb pipeline list

# Search for repositories
bb search repos "api" --workspace myworkspace
```

## Commands

### Core Commands
- `bb auth` - Authenticate bb and git with Bitbucket
- `bb repo` - Manage repositories (list, clone, create, view, edit, delete, sync, fork)
- `bb pr` - Manage pull requests (list, view, create, checkout, merge, close, diff, checks)
- `bb issue` - Manage issues (list, view, create, edit, close, reopen, comment)

### Additional Commands
- `bb pipeline` - Manage pipelines (list, view, run, cancel)
- `bb project` - Work with Bitbucket projects (list, view, create)
- `bb workspace` - Manage workspaces (list, view)
- `bb search` - Search for repositories and code
- `bb ssh-key` - Manage SSH keys (list, add, delete)
- `bb secret` - Manage repository secrets
- `bb variable` - Manage pipeline variables
- `bb browse` - Open repository in browser
- `bb api` - Make authenticated API requests
- `bb status` - View status across workspaces
- `bb config` - Manage configuration
- `bb alias` - Create command shortcuts

## Configuration

Configuration is stored in `~/.config/bb/` (Linux/macOS) or `%APPDATA%\bb\` (Windows).

```bash
# View current configuration
bb config list

# Set a configuration value
bb config set editor vim

# Get a configuration value
bb config get editor
```

## Shell Completion

```bash
# Bash
bb completion -s bash > /etc/bash_completion.d/bb

# Zsh
bb completion -s zsh > "${fpath[1]}/_bb"

# Fish
bb completion -s fish > ~/.config/fish/completions/bb.fish

# PowerShell
bb completion -s powershell >> $PROFILE
```

## Environment Variables

- `BB_TOKEN` - Authentication token (alternative to `bb auth login`)
- `BB_HOST` - Bitbucket host (default: bitbucket.org)
- `BB_REPO` - Default repository in `WORKSPACE/REPO` format
- `BB_PAGER` - Pager program (default: less)
- `NO_COLOR` - Disable colored output

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

This project is based on the excellent [GitHub CLI](https://github.com/cli/cli) architecture.
