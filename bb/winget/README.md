# WinGet Package for Bitbucket CLI

This directory contains the Windows Package Manager (WinGet) manifest files for the Bitbucket CLI.

## Installation (after package is published)

```powershell
winget install dlbroadfoot.bb
```

## Submitting to WinGet Repository

### First-time Submission

1. Fork the [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) repository
2. Copy the manifest files from `manifests/d/dlbroadfoot/bb/<version>/` to your fork
3. Create a pull request to the upstream repository

### Updating for New Releases

You can use [wingetcreate](https://github.com/microsoft/winget-create) to automate updates:

```powershell
# Install wingetcreate
winget install Microsoft.WingetCreate

# Update manifest for a new version
wingetcreate update dlbroadfoot.bb --version <NEW_VERSION> \
  --urls https://github.com/dlbroadfoot/bitbucket-cli/releases/download/v<NEW_VERSION>/bb_<NEW_VERSION>_windows_386.msi \
         https://github.com/dlbroadfoot/bitbucket-cli/releases/download/v<NEW_VERSION>/bb_<NEW_VERSION>_windows_amd64.msi \
         https://github.com/dlbroadfoot/bitbucket-cli/releases/download/v<NEW_VERSION>/bb_<NEW_VERSION>_windows_arm64.msi \
  --submit
```

### GitHub Actions Integration

You can add this to your release workflow to auto-submit updates:

```yaml
- name: Update WinGet manifest
  uses: vedantmgoyal9/winget-releaser@main
  with:
    identifier: dlbroadfoot.bb
    installers-regex: '\.msi$'
    token: ${{ secrets.WINGET_PAT }}
```

Note: Requires a GitHub Personal Access Token with `public_repo` scope.

## Manifest Files

- `dlbroadfoot.bb.yaml` - Version manifest
- `dlbroadfoot.bb.installer.yaml` - Installer URLs and checksums
- `dlbroadfoot.bb.locale.en-US.yaml` - Package description and metadata

## Validating Manifests

```powershell
winget validate manifests/d/dlbroadfoot/bb/<version>/
```
