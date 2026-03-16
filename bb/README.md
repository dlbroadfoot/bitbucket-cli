# bb

This directory contains the source code for the Bitbucket CLI (`bb`).

See the [top-level README](../README.md) for installation instructions and usage documentation.

## Development

### Prerequisites

- Go 1.21+

### Build

```sh
make bin/bb
```

### Test

```sh
make test
```

### Release

Releases are built automatically via GitHub Actions when a tag is pushed:

```sh
git tag v2.x.x
git push origin v2.x.x
```

This produces binaries for macOS (universal), Linux (amd64, arm64, 386, armv6), and Windows (amd64, arm64, 386), along with `.deb`, `.rpm`, `.pkg`, and `.msi` installers.

### Project structure

```text
bb/
  cmd/bb/        Entry point
  pkg/cmd/       Command implementations (Cobra)
  api/           Bitbucket REST API client
  internal/      Internal packages (config, auth, repo model)
  build/         Installer build configs (macOS PKG, Windows MSI)
  script/        Build and release scripts
  winget/        WinGet package manifests
```
