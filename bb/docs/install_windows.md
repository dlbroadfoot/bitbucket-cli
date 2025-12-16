# Installing bb on Windows

## Precompiled binaries

[Bitbucket CLI releases](https://github.com/dlbroadfoot/bitbucket-cli/releases/latest) contain precompiled `exe` and `msi` binaries for `386`, `amd64` and `arm64` architectures.

### Using the MSI installer (Recommended)

Download the appropriate MSI installer for your architecture:
- `bb_X.Y.Z_windows_amd64.msi` for 64-bit Windows
- `bb_X.Y.Z_windows_386.msi` for 32-bit Windows
- `bb_X.Y.Z_windows_arm64.msi` for ARM64 Windows

Double-click the MSI file to install. The installer will:
- Install `bb.exe` to `C:\Program Files\Bitbucket CLI\`
- Add the install directory to your system PATH

> [!NOTE]
> The Windows installer modifies your PATH. When using Windows Terminal, you will need to **open a new window** for the changes to take effect. (Simply opening a new tab will _not_ be sufficient.)

### Using the zip archive

1. Download the appropriate zip file for your architecture:
   - `bb_X.Y.Z_windows_amd64.zip` for 64-bit Windows
   - `bb_X.Y.Z_windows_386.zip` for 32-bit Windows
   - `bb_X.Y.Z_windows_arm64.zip` for ARM64 Windows

2. Extract the archive

3. Add the extracted directory to your PATH, or move `bb.exe` to a directory already in your PATH

## Building from source

See [install_source.md](install_source.md) for instructions on building from source.
