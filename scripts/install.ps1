$ErrorActionPreference = "Stop"

$Repo = if ($env:GROKBUILD_REPO) {
    $env:GROKBUILD_REPO
} else {
    "GreyGunG/grokbuild-proxy"
}
$Version = if ($env:GROKBUILD_VERSION) {
    $env:GROKBUILD_VERSION
} else {
    "latest"
}
$InstallDir = if ($env:INSTALL_DIR) {
    $env:INSTALL_DIR
} else {
    Join-Path $HOME ".local\bin"
}
$ConfigDir = if ($env:CONFIG_DIR) {
    $env:CONFIG_DIR
} else {
    Join-Path $HOME ".config\grokbuild-proxy"
}

$Architecture = $env:PROCESSOR_ARCHITECTURE
switch ($Architecture.ToUpperInvariant()) {
    "AMD64" { $Arch = "x86_64" }
    "ARM64" { $Arch = "arm64" }
    default { throw "Unsupported architecture: $Architecture" }
}

$Archive = "grokbuild-proxy_Windows_$Arch.zip"
if ($Version -eq "latest") {
    $BaseUrl = "https://github.com/$Repo/releases/latest/download"
} else {
    $BaseUrl = "https://github.com/$Repo/releases/download/$Version"
}

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) (
    "grokbuild-proxy-" + [System.Guid]::NewGuid().ToString("N")
)
New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
    $ArchivePath = Join-Path $TempDir $Archive
    $ChecksumsPath = Join-Path $TempDir "checksums.txt"

    Write-Host "Downloading $Archive..."
    Invoke-WebRequest -UseBasicParsing `
        -Uri "$BaseUrl/$Archive" `
        -OutFile $ArchivePath
    Invoke-WebRequest -UseBasicParsing `
        -Uri "$BaseUrl/checksums.txt" `
        -OutFile $ChecksumsPath

    $ChecksumLine = Get-Content $ChecksumsPath |
        Where-Object { $_ -match "\s+$([regex]::Escape($Archive))$" } |
        Select-Object -First 1
    if (-not $ChecksumLine) {
        throw "Checksum entry not found for $Archive"
    }

    $Expected = ($ChecksumLine -split "\s+")[0].ToLowerInvariant()
    $Actual = (Get-FileHash -Algorithm SHA256 $ArchivePath).Hash.ToLowerInvariant()
    if ($Expected -ne $Actual) {
        throw "Checksum verification failed"
    }

    $ExtractDir = Join-Path $TempDir "extracted"
    Expand-Archive -Path $ArchivePath -DestinationPath $ExtractDir

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null

    $BinarySource = Join-Path $ExtractDir "grokbuild-proxy.exe"
    $BinaryTarget = Join-Path $InstallDir "grokbuild-proxy.exe"
    Copy-Item -Force $BinarySource $BinaryTarget

    $ConfigTarget = Join-Path $ConfigDir "config.yaml"
    if (-not (Test-Path $ConfigTarget)) {
        Copy-Item `
            (Join-Path $ExtractDir "config.example.yaml") `
            $ConfigTarget
    }

    Write-Host ""
    Write-Host "Installed: $BinaryTarget"
    Write-Host "Config:    $ConfigTarget"
    Write-Host ""
    Write-Host "Run:"
    Write-Host "  `"$BinaryTarget`" -config `"$ConfigTarget`""
    Write-Host ""
    Write-Host "Add $InstallDir to PATH to run grokbuild-proxy directly."
} finally {
    Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}
