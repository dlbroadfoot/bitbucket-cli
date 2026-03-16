# bb - Bitbucket CLI

`bb` brings Bitbucket to your terminal. Manage pull requests, repositories, pipelines, and more — right next to where you're already working with `git`.

## Install

**macOS, Linux, and WSL:**

```sh
curl -fsSL https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/install.ps1 | iex
```

### Other install methods

<details>
<summary>WinGet (Windows)</summary>

```powershell
winget install dlbroadfoot.bb
```

Note: WinGet may lag behind the latest release.
</details>

<details>
<summary>Homebrew (macOS/Linux)</summary>

```sh
brew install dlbroadfoot/tap/bb
```
</details>

<details>
<summary>Go install</summary>

```sh
go install github.com/dlbroadfoot/bitbucket-cli/bb/cmd/bb@latest
```

Installs to `$GOPATH/bin` (typically `~/go/bin`).
</details>

<details>
<summary>Debian / Ubuntu (.deb)</summary>

Download the `.deb` from the [latest release](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest):

```sh
sudo dpkg -i bb_*_linux_amd64.deb
```
</details>

<details>
<summary>Fedora / RHEL (.rpm)</summary>

Download the `.rpm` from the [latest release](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest):

```sh
sudo rpm -i bb_*_linux_amd64.rpm
```
</details>

<details>
<summary>macOS PKG installer</summary>

Download the `.pkg` from the [latest release](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest):

```sh
sudo installer -pkg bb_*_macOS_universal.pkg -target /
```
</details>

<details>
<summary>Windows MSI installer</summary>

Download the `.msi` from the [latest release](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest) and run it.
</details>

<details>
<summary>Manual download</summary>

Pre-built binaries for all platforms are available on the [releases page](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest).
</details>

## Get started

```sh
# Authenticate with Bitbucket (opens browser to guide you through token creation)
bb auth login --web

# List pull requests in the current repo
bb pr list

# Create a pull request
bb pr create --title "My change" --body "Description"

# Clone a repository
bb repo clone myworkspace/myrepo

# View pipeline runs
bb pipeline list
```

## Commands

### Core

| Command | Description |
|---------|-------------|
| `bb auth` | Authenticate with Bitbucket |
| `bb pr` | Create, view, list, checkout, merge, close pull requests |
| `bb issue` | Create, view, list, edit, close, reopen issues |
| `bb repo` | Clone, create, fork, view, list, manage repositories |

### CI/CD & Projects

| Command | Description |
|---------|-------------|
| `bb pipeline` | View, run, and cancel pipeline builds |
| `bb project` | List and view Bitbucket projects |
| `bb workspace` | List and view workspaces |
| `bb secret` | Manage repository secrets |
| `bb variable` | Manage pipeline variables |

### Utilities

| Command | Description |
|---------|-------------|
| `bb browse` | Open repository in browser |
| `bb search` | Search repositories and code |
| `bb api` | Make authenticated API requests |
| `bb ssh-key` | Manage SSH keys |
| `bb status` | View status across workspaces |
| `bb config` | Manage configuration |
| `bb alias` | Create command shortcuts |

Run `bb <command> --help` for details on any command.

## Authentication

`bb` uses Bitbucket [API tokens](https://id.atlassian.com/manage-profile/security/api-tokens) for authentication.

```sh
# Guided setup — opens browser to create a token with step-by-step instructions
bb auth login --web

# Or authenticate non-interactively
BB_TOKEN=email:token bb pr list
```

**Required scopes:** Account (Read), Repositories (Read/Write), Pull Requests (Read/Write). Add Issues and Pipelines permissions if you use those features.

## Configuration

Config is stored in `~/.config/bb/` (Linux/macOS) or `%APPDATA%\bb\` (Windows).

```sh
bb config list           # View config
bb config set editor vim # Set a value
```

**Environment variables:** `BB_TOKEN`, `BB_HOST`, `BB_REPO`, `BB_PAGER`, `NO_COLOR`

## Shell completion

```sh
# Bash
bb completion -s bash > /etc/bash_completion.d/bb

# Zsh
bb completion -s zsh > "${fpath[1]}/_bb"

# Fish
bb completion -s fish > ~/.config/fish/completions/bb.fish

# PowerShell
bb completion -s powershell >> $PROFILE
```

## License

MIT - see [LICENSE](bb/LICENSE) for details.

## Acknowledgments

Built on the architecture of the [GitHub CLI](https://github.com/cli/cli).
