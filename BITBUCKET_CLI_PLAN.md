# Bitbucket CLI (`bb`) Implementation Plan

## Overview

Create a Bitbucket CLI tool by adapting the GitHub CLI (`gh`) codebase. The tool will be called `bb` and provide equivalent functionality for Bitbucket Cloud.

## Decisions

- **Authentication**: App Passwords (Bitbucket's PAT equivalent) for initial implementation, OAuth later
- **Command scope**: Focus on `auth`, `pr`, `issue` commands (most common AI agent use cases)
- **Target**: Bitbucket Cloud only (no Data Center support initially)
- **Repository structure**: Reorganize into `gh/` and `bb/` subfolders at root level

## Key Technical Findings

### GitHub CLI Architecture (Reusable)
- **Entry point**: `cmd/gh/main.go`
- **Commands**: `pkg/cmd/` (36+ command directories using Cobra framework)
- **Factory pattern**: `pkg/cmdutil/Factory` for dependency injection
- **Keyring**: `internal/keyring/keyring.go` using `zalando/go-keyring` (supports Windows, macOS, Linux)
- **Config**: YAML-based at `~/.config/gh/`

### Critical Bitbucket Difference: No Device Flow
- **GitHub**: Uses OAuth Device Flow (shows code, user enters on website)
- **Bitbucket**: Only supports Authorization Code Grant (local callback server required)
- The `cli/oauth` library already supports web app flow with local callback, so this is workable

### Bitbucket OAuth Details
- **Auth URL**: `https://bitbucket.org/site/oauth2/authorize`
- **Token URL**: `https://bitbucket.org/site/oauth2/access_token`
- **Token expiry**: 2 hours (requires refresh token handling)
- **Scopes**: Defined at consumer registration, not per-request
- **Key scopes needed**: `repository`, `pullrequest:write`, `account`

### Bitbucket API
- **Base URL**: `https://api.bitbucket.org/2.0/`
- **REST only** (no GraphQL)
- **Pagination**: `next` URL in response body
- **Auth header**: `Bearer {token}` (not `token` prefix like GitHub)
- **Terminology**: `workspace/repo_slug` instead of `owner/repo`

---

## Repository Structure

Reorganize the codebase into two parallel tools:

```
lagos/
├── gh/                    # Original GitHub CLI (moved from root)
│   ├── cmd/gh/
│   ├── pkg/
│   ├── api/
│   ├── internal/
│   ├── go.mod
│   └── ...
├── bb/                    # New Bitbucket CLI
│   ├── cmd/bb/
│   ├── pkg/
│   ├── api/
│   ├── internal/
│   ├── go.mod
│   └── ...
└── README.md
```

---

## Implementation Phases

### Phase 1: Repository Reorganization ✅

**1.1 Move gh to subfolder**
- Create `gh/` directory
- Move all existing source files into `gh/`
- Update `gh/go.mod` module path if needed

**1.2 Create bb scaffolding**
- Copy `gh/` to `bb/`
- Initial rename pass: `gh` → `bb`, `GH_` → `BB_`, `GitHub` → `Bitbucket`

### Phase 2: Foundation (bb-specific) ✅

**2.1 Instance Management**
- Create `bb/internal/bbinstance/bbinstance.go`
- Define Cloud endpoint: `https://api.bitbucket.org/2.0/`
- Remove Data Center/Enterprise logic

**2.2 Repository Model**
- Create `bb/internal/bbrepo/repo.go`
- Change `owner` → `workspace`, `repo` → `repo_slug`

**Files to create/modify:**
- `bb/cmd/bb/main.go`
- `bb/go.mod` (new module path)
- `bb/internal/bbinstance/bbinstance.go`
- `bb/internal/bbrepo/repo.go`
- `bb/Makefile`

### Phase 3: Authentication (App Passwords) ✅

**3.1 App Password Login**
App Passwords are Bitbucket's equivalent of GitHub PATs. User creates in Bitbucket settings, then:

```
bb auth login
? Bitbucket username: myuser
? App password: ****
```

**3.2 Token Storage**
- Store username + app password in keyring
- Service name: `bb:bitbucket.org`
- No refresh tokens needed (app passwords don't expire)

**3.3 Login Command**
- Simplify `bb/pkg/cmd/auth/login/login.go`
- Prompt for username and app password
- Verify credentials by calling `GET /user`
- Store in keyring on success

**Files to modify:**
- `bb/internal/authflow/` - Simplify or replace with app password logic
- `bb/internal/config/config.go` - Token storage
- `bb/api/http_client.go` - Basic Auth with username:app_password
- `bb/pkg/cmd/auth/login/login.go`
- `bb/pkg/cmd/auth/logout/logout.go`
- `bb/pkg/cmd/auth/status/status.go`

### Phase 4: API Client 🔄

**4.1 REST Client**
- Create `bb/api/client.go` - REST-only, remove GraphQL
- Authentication: Basic Auth header with `username:app_password`
- Handle Bitbucket pagination (`next` field in response)

**4.2 API Types**
- Create `bb/api/types.go`
- Define: User, Workspace, Repository, PullRequest, Issue, Branch, Commit

**Files to create:**
- `bb/api/client.go`
- `bb/api/types.go`

### Phase 5: Core Commands

**Commands to implement (priority order):**

| Command | Bitbucket Endpoint | Notes |
|---------|-------------------|-------|
| `bb auth login` | App password entry | Foundation |
| `bb auth logout` | Clear keyring | |
| `bb auth status` | GET /user | Verify auth |
| `bb pr list` | GET /repositories/{workspace}/{repo}/pullrequests | |
| `bb pr view` | GET /repositories/{workspace}/{repo}/pullrequests/{id} | |
| `bb pr create` | POST /repositories/{workspace}/{repo}/pullrequests | |
| `bb pr merge` | POST /repositories/{workspace}/{repo}/pullrequests/{id}/merge | |
| `bb pr checkout` | git fetch + checkout | |
| `bb issue list` | GET /repositories/{workspace}/{repo}/issues | |
| `bb issue view` | GET /repositories/{workspace}/{repo}/issues/{id} | |
| `bb issue create` | POST /repositories/{workspace}/{repo}/issues | |
| `bb project list` | GET /workspaces/{workspace}/projects | List projects in workspace |
| `bb project view` | GET /workspaces/{workspace}/projects/{project_key} | View project details |
| `bb project create` | POST /workspaces/{workspace}/projects | Create new project |
| `bb api` | Raw API access | For flexibility |

**Commands to remove:**
- `codespace`, `gist`, `actions`, `workflow`, `run`, `attestation`, `ruleset`, `release`, `cache`, `variable`, `secret`, `gpg-key`, `ssh-key`
- `label` - No native label support in Bitbucket Cloud (Server/Data Center only)

### Phase 6: Configuration

- Config directory: `~/.config/bb/`
- Environment variables: `BB_TOKEN` (app password), `BB_HOST`, `BB_REPO`
- Keyring service: `bb:bitbucket.org`

---

## App Password Authentication Details

**How App Passwords work:**
1. User creates App Password at `https://bitbucket.org/account/settings/app-passwords/`
2. User selects permissions (scopes): Repositories, Pull requests, Issues, etc.
3. Bitbucket generates a password string
4. CLI stores `username:app_password` and uses Basic Auth

**Required permissions for bb CLI:**
- Account: Read
- Repositories: Read, Write
- Pull requests: Read, Write
- Issues: Read, Write

**API Authentication:**
```
Authorization: Basic base64(username:app_password)
```

**Git Authentication:**
```
git clone https://username:app_password@bitbucket.org/workspace/repo.git
```

---

## Bitbucket API Key Differences from GitHub

| Aspect | GitHub | Bitbucket |
|--------|--------|-----------|
| Terminology | owner/repo | workspace/repo_slug |
| PR states | open, closed, merged | OPEN, MERGED, DECLINED, SUPERSEDED |
| API style | REST + GraphQL | REST only |
| Pagination | Link header | `next` URL in response body |
| Auth header | `token {pat}` | `Basic base64(user:pass)` |
| Draft PRs | Yes | No |

---

## Critical Files Reference

| Source (gh/) | Target (bb/) | Changes Needed |
|-------------|-------------|----------------|
| `internal/ghinstance/` | `internal/bbinstance/` | Bitbucket Cloud URL only |
| `internal/ghrepo/` | `internal/bbrepo/` | workspace/repo_slug |
| `internal/authflow/` | `internal/authflow/` | App password (simpler) |
| `internal/config/` | `internal/config/` | Service name `bb:` |
| `api/client.go` | `api/client.go` | REST only, Basic Auth |
| `pkg/cmd/auth/` | `pkg/cmd/auth/` | App password login |
| `pkg/cmd/pr/` | `pkg/cmd/pr/` | Bitbucket PR endpoints |
| `pkg/cmd/issue/` | `pkg/cmd/issue/` | Bitbucket issue endpoints |
| `pkg/cmd/project/` | `pkg/cmd/project/` | Bitbucket project endpoints (workspace-level) |
