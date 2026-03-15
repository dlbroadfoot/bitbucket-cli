# Bitbucket CLI (bb) installer for Windows
# Usage: irm https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "dlbroadfoot/bitbucket-cli"

# Detect architecture
$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    "x86"   { "386" }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
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

# Install MSI
Write-Host "Installing..." -ForegroundColor Cyan
Start-Process msiexec.exe -ArgumentList "/i", "`"$MsiPath`"", "/quiet", "/norestart" -Wait -Verb RunAs

# Cleanup
Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue

# Verify
$BbPath = Get-Command bb -ErrorAction SilentlyContinue
if ($BbPath) {
    Write-Host "`nBitbucket CLI (bb) $Version installed successfully!" -ForegroundColor Green
    Write-Host "`nGet started:"
    Write-Host "  bb auth login    # Authenticate with Bitbucket"
    Write-Host "  bb --help        # See all commands"
} else {
    Write-Host "`nInstalled bb $Version. You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
}
