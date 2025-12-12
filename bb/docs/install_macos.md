# Installing bb on macOS

## Precompiled binaries

[Bitbucket CLI releases](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest) contain precompiled binaries for `amd64` and `arm64` architectures along with a universal `.pkg` installer.

### Using the .pkg installer

Download `bb_X.Y.Z_macOS_universal.pkg` from the releases page and double-click to install. This will:
- Install `bb` to `/usr/local/bin`
- Add shell completions for zsh
- Add man pages

### Using the zip archive

1. Download the appropriate zip file for your architecture:
   - `bb_X.Y.Z_macOS_amd64.zip` for Intel Macs
   - `bb_X.Y.Z_macOS_arm64.zip` for Apple Silicon Macs

2. Extract the archive:
   ```shell
   unzip bb_X.Y.Z_macOS_arm64.zip
   ```

3. Move the binary to a directory in your PATH:
   ```shell
   sudo mv bb_X.Y.Z_macOS_arm64/bb /usr/local/bin/
   ```

## Building from source

See [install_source.md](install_source.md) for instructions on building from source.
