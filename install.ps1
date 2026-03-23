# Bitbucket CLI (bb) installer for Windows
# Usage: irm https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "dlbroadfoot/bitbucket-cli"

# Detect architecture (account for WoW64 — 32-bit PowerShell on 64-bit OS)
$RawArch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
$Arch = switch ($RawArch) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    "x86"   { "386" }
    default { throw "Unsupported architecture: $RawArch" }
}

Write-Host "Installing Bitbucket CLI (bb)..." -ForegroundColor Cyan
Write-Host "  Arch: $Arch"

# Get latest release
Write-Host "Fetching latest release..." -ForegroundColor Cyan
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name.TrimStart("v")
Write-Host "  Version: $Version"

# Download MSI
$Filename = "bb_${Version}_windows_${Arch}.msi"
$Url = "https://github.com/$Repo/releases/download/v${Version}/$Filename"
$TempDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "bb-install-$(Get-Random)")
$MsiPath = Join-Path $TempDir $Filename

Write-Host "Downloading $Filename..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $Url -OutFile $MsiPath -UseBasicParsing

# Verify checksum
Write-Host "Verifying checksum..." -ForegroundColor Cyan
$ChecksumsUrl = "https://github.com/$Repo/releases/download/v${Version}/checksums.txt"
$ChecksumsPath = Join-Path $TempDir "checksums.txt"
Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath -UseBasicParsing

$ExpectedHash = (
    Select-String -Path $ChecksumsPath -Pattern ([regex]::Escape($Filename)) |
    Select-Object -First 1
).Line.Split()[0]
if (-not $ExpectedHash) { throw "Could not find checksum for $Filename" }

$ActualHash = (Get-FileHash -Path $MsiPath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($ActualHash -ne $ExpectedHash.ToLowerInvariant()) {
    throw "Checksum mismatch for $Filename (expected: $ExpectedHash, got: $ActualHash)"
}

# Install MSI
Write-Host "Installing..." -ForegroundColor Cyan
$MsiProcess = Start-Process msiexec.exe `
    -ArgumentList "/i", "`"$MsiPath`"", "/quiet", "/norestart" `
    -Wait -PassThru -Verb RunAs

if ($MsiProcess.ExitCode -ne 0) {
    throw "MSI installation failed with exit code $($MsiProcess.ExitCode)"
}

# Cleanup
Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue

# Verify
$BbPath = Get-Command bb -ErrorAction SilentlyContinue
if ($BbPath) {
    Write-Host "`nBitbucket CLI (bb) $Version installed successfully!" -ForegroundColor Green
    Write-Host "`nGet started:"
    Write-Host "  bb auth login --web    # Authenticate with Bitbucket (opens browser for token creation)"
    Write-Host "  bb --help              # See all commands"
} else {
    Write-Host "`nInstalled bb $Version. You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
    Write-Host "  bb auth login --web"
}
