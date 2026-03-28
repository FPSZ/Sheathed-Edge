param(
    [string]$ProjectRoot = "D:\AI\Local",
    [string]$WslRoot = "D:\Environment2\WSL",
    [string]$DistroName = "Ubuntu-24.04",
    [string]$LinuxUser = "awdp",
    [string]$UbuntuRootfsUrl = "https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-amd64-root.tar.xz",
    [string]$WslMsiUrl = "https://github.com/microsoft/WSL/releases/download/2.6.3/wsl.2.6.3.0.x64.msi",
    [switch]$SkipFeatureInstall,
    [switch]$SkipDownload,
    [switch]$SkipImport,
    [switch]$SkipBootstrap
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Test-Admin {
    $principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Ensure-WslRuntimeInstalled {
    param(
        [string]$ImagesDir,
        [string]$MsiUrl
    )

    Write-Step "Ensure WSL runtime is installed"

    & wsl.exe --status 2>&1 | Out-Host
    if ($LASTEXITCODE -eq 0) {
        Write-Host "WSL runtime is already available"
        return
    }

    if (-not (Test-Admin)) {
        throw "WSL runtime is not available yet, and installing it requires Administrator privileges. Re-run this script from an elevated PowerShell session."
    }

    Write-Host "Trying online runtime install first: wsl.exe --install --no-distribution --web-download"
    & wsl.exe --install --no-distribution --web-download
    $installExitCode = $LASTEXITCODE

    if (($installExitCode -ne 0) -and ($installExitCode -ne 3010)) {
        Write-Host "Online runtime install did not succeed. Falling back to inbox runtime install."
        & wsl.exe --install --no-distribution --inbox
        $installExitCode = $LASTEXITCODE
    }

    if (($installExitCode -ne 0) -and ($installExitCode -ne 3010)) {
        $msiName = Split-Path -Leaf $MsiUrl
        $msiPath = Join-Path $ImagesDir $msiName

        if (-not (Test-Path -LiteralPath $msiPath)) {
            Write-Host "Downloading WSL MSI from official GitHub release: $MsiUrl"
            Invoke-WebRequest -Uri $MsiUrl -OutFile $msiPath
        } else {
            Write-Host "Using cached WSL MSI: $msiPath"
        }

        Write-Host "Installing WSL MSI via msiexec"
        $msiProcess = Start-Process -FilePath "msiexec.exe" -ArgumentList @("/i", "`"$msiPath`"", "/passive", "/norestart") -Wait -PassThru
        $installExitCode = $msiProcess.ExitCode
    }

    if (($installExitCode -ne 0) -and ($installExitCode -ne 3010)) {
        throw "WSL runtime installation failed for web-download, inbox, and MSI fallback modes. Exit code: $installExitCode"
    }

    if ($installExitCode -eq 3010) {
        throw "WSL runtime installation succeeded but requires another reboot. Reboot Windows and rerun this script with -SkipFeatureInstall."
    }

    & wsl.exe --status 2>&1 | Out-Host
    if ($LASTEXITCODE -ne 0) {
        throw "WSL runtime still is not available after installation. Check Microsoft Store / web-download policy, then rerun."
    }
}

function Write-WslConfig {
    param([string]$WslRootPath)

    $wslConfigPath = Join-Path $env:USERPROFILE ".wslconfig"
    $content = @"
[wsl2]
memory=24GB
processors=12
swap=8GB
swapFile=D:\\Environment2\\WSL\\swap.vhdx
localhostForwarding=true
networkingMode=nat
dnsTunneling=true
firewall=true
autoProxy=false
"@

    $normalized = $content.Trim() + "`r`n"
    $existing = if (Test-Path -LiteralPath $wslConfigPath) {
        Get-Content -LiteralPath $wslConfigPath -Raw -Encoding UTF8
    } else {
        ""
    }

    if ($existing -ne $normalized) {
        Set-Content -LiteralPath $wslConfigPath -Value $normalized -Encoding UTF8
        Write-Host "Wrote $wslConfigPath"
    } else {
        Write-Host "$wslConfigPath already matches target config"
    }
}

function Enable-WindowsWslFeatures {
    if (-not (Test-Admin)) {
        throw "Administrator privileges are required to enable WSL features. Re-run this script from an elevated PowerShell session."
    }

    $rebootRequired = $false

    Write-Step "Enable Windows WSL features"
    & dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart
    if ($LASTEXITCODE -eq 3010) {
        $rebootRequired = $true
    } elseif ($LASTEXITCODE -ne 0) {
        throw "Failed to enable Microsoft-Windows-Subsystem-Linux. Exit code: $LASTEXITCODE"
    }

    & dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
    if ($LASTEXITCODE -eq 3010) {
        $rebootRequired = $true
    } elseif ($LASTEXITCODE -ne 0) {
        throw "Failed to enable VirtualMachinePlatform. Exit code: $LASTEXITCODE"
    }

    if ($rebootRequired) {
        Write-Host "Windows features were enabled and a reboot is required. Reboot Windows first, then rerun this script with -SkipFeatureInstall."
    } else {
        Write-Host "Windows features were enabled. Rerun this script with -SkipFeatureInstall."
    }
    exit 0
}

function Get-DistroState {
    param([string]$Name)

    $list = & wsl.exe -l -v 2>&1
    if ($LASTEXITCODE -ne 0) {
        return $null
    }

    foreach ($line in $list) {
        if ($line -match "^\s*\*?\s*$([regex]::Escape($Name))\s+") {
            return $line
        }
    }

    return $null
}

function Invoke-Download {
    param(
        [string]$Url,
        [string]$Destination
    )

    Write-Step "Download Ubuntu rootfs"
    Invoke-WebRequest -Uri $Url -OutFile $Destination

    $shaUrl = ($Url.Substring(0, $Url.LastIndexOf("/") + 1)) + "SHA256SUMS"
    $shaPath = Join-Path (Split-Path -Parent $Destination) "SHA256SUMS"
    Invoke-WebRequest -Uri $shaUrl -OutFile $shaPath

    $targetName = Split-Path -Leaf $Destination
    $expectedLine = Select-String -Path $shaPath -Pattern ([regex]::Escape($targetName)) | Select-Object -First 1
    if (-not $expectedLine) {
        throw "Could not find $targetName in SHA256SUMS"
    }

    $expectedHash = ($expectedLine.Line -split "\s+")[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -LiteralPath $Destination -Algorithm SHA256).Hash.ToLowerInvariant()

    if ($expectedHash -ne $actualHash) {
        throw "Rootfs checksum mismatch. expected=$expectedHash actual=$actualHash"
    }

    Write-Host "Rootfs checksum verified"
}

function Import-Distro {
    param(
        [string]$Name,
        [string]$InstallPath,
        [string]$ArchivePath
    )

    if (Get-DistroState -Name $Name) {
        Write-Host "$Name already exists, skipping import"
        return
    }

    Write-Step "Import $Name to $InstallPath"
    & wsl.exe --import $Name $InstallPath $ArchivePath --version 2
    if ($LASTEXITCODE -ne 0) {
        throw "wsl --import failed. Exit code: $LASTEXITCODE"
    }
}

function Bootstrap-Distro {
    param(
        [string]$Name,
        [string]$RepoRoot,
        [string]$UserName
    )

    $scriptPath = "/mnt/d/AI/Local/Workflows/wsl/bootstrap-ubuntu.sh"
    Write-Step "Run bootstrap script inside WSL"
    & wsl.exe -d $Name -u root -- bash $scriptPath $UserName $RepoRoot
    if ($LASTEXITCODE -ne 0) {
        throw "bootstrap-ubuntu.sh failed. Exit code: $LASTEXITCODE"
    }

    & wsl.exe --terminate $Name | Out-Null
}

$distrosDir = Join-Path $WslRoot "Distros"
$imagesDir = Join-Path $WslRoot "Images"
$exportDir = Join-Path $WslRoot "Export"
$installPath = Join-Path $distrosDir $DistroName
$downloadName = Split-Path -Leaf $UbuntuRootfsUrl
$downloadPath = Join-Path $imagesDir $downloadName
$importArchivePath = $downloadPath
$repoRootWsl = "/mnt/d/AI/Local"

Ensure-Directory -Path $WslRoot
Ensure-Directory -Path $distrosDir
Ensure-Directory -Path $imagesDir
Ensure-Directory -Path $exportDir

Write-WslConfig -WslRootPath $WslRoot

if (-not $SkipFeatureInstall) {
    Enable-WindowsWslFeatures
}

Ensure-WslRuntimeInstalled -ImagesDir $imagesDir -MsiUrl $WslMsiUrl

if (-not $SkipDownload) {
    Invoke-Download -Url $UbuntuRootfsUrl -Destination $downloadPath
}

if (-not $SkipImport) {
    Import-Distro -Name $DistroName -InstallPath $installPath -ArchivePath $importArchivePath
}

if (-not $SkipBootstrap) {
    Bootstrap-Distro -Name $DistroName -RepoRoot $repoRootWsl -UserName $LinuxUser
}

Write-Step "Done"
Write-Host "Next steps:"
Write-Host "1. Run .\\Workflows\\wsl\\validate-wsl.ps1 to verify the environment"
$exportTarget = Join-Path $exportDir "ubuntu-24.04-baseline.tar"
Write-Host ('2. Export a clean snapshot: wsl.exe --export {0} "{1}"' -f $DistroName, $exportTarget)
