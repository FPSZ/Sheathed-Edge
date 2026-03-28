param(
    [string]$InstallRoot = "D:\Environment2\Create\llama.cpp",
    [string]$PreferredFlavor = "hip-radeon",
    [switch]$Force
)

$ErrorActionPreference = "Stop"

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Select-AssetName {
    param([string]$Flavor)

    switch ($Flavor) {
        "hip-radeon" { return @("llama-*-bin-win-hip-radeon-x64.zip", "llama-*-bin-win-vulkan-x64.zip", "llama-*-bin-win-cpu-x64.zip") }
        "vulkan" { return @("llama-*-bin-win-vulkan-x64.zip", "llama-*-bin-win-hip-radeon-x64.zip", "llama-*-bin-win-cpu-x64.zip") }
        "cpu" { return @("llama-*-bin-win-cpu-x64.zip") }
        default { throw "Unsupported flavor: $Flavor" }
    }
}

Ensure-Directory -Path $InstallRoot
$downloadRoot = Join-Path $InstallRoot "_downloads"
$extractRoot = Join-Path $InstallRoot "_extract"
Ensure-Directory -Path $downloadRoot
Ensure-Directory -Path $extractRoot

Write-Host "Resolving latest llama.cpp Windows release"
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/ggml-org/llama.cpp/releases/latest"
$patterns = Select-AssetName -Flavor $PreferredFlavor
$asset = $null
foreach ($pattern in $patterns) {
    $asset = $release.assets | Where-Object { $_.name -like $pattern } | Select-Object -First 1
    if ($asset) { break }
}
if (-not $asset) {
    throw "No matching Windows asset found for flavor $PreferredFlavor"
}

$zipPath = Join-Path $downloadRoot $asset.name
if ($Force -or -not (Test-Path -LiteralPath $zipPath)) {
    Write-Host "Downloading $($asset.browser_download_url)"
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath
}

$stagingDir = Join-Path $extractRoot $release.tag_name
if (Test-Path -LiteralPath $stagingDir) {
    Remove-Item -LiteralPath $stagingDir -Recurse -Force
}
Ensure-Directory -Path $stagingDir

Write-Host "Extracting $zipPath"
Expand-Archive -LiteralPath $zipPath -DestinationPath $stagingDir -Force

$exe = Get-ChildItem -LiteralPath $stagingDir -Recurse -Filter llama-server.exe -ErrorAction Stop | Select-Object -First 1
if (-not $exe) {
    throw "llama-server.exe not found after extraction"
}

Write-Host "Syncing extracted files into $InstallRoot"
Get-ChildItem -LiteralPath $stagingDir -Force | ForEach-Object {
    $destination = Join-Path $InstallRoot $_.Name
    if ($_.PSIsContainer) {
        Copy-Item -LiteralPath $_.FullName -Destination $destination -Recurse -Force
    } else {
        Copy-Item -LiteralPath $_.FullName -Destination $destination -Force
    }
}

$resolvedExe = Get-ChildItem -LiteralPath $InstallRoot -Recurse -Filter llama-server.exe -ErrorAction Stop | Select-Object -First 1
Write-Host "Installed llama-server: $($resolvedExe.FullName)"
