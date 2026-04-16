[CmdletBinding()]
param(
    [string]$Username = $env:USERNAME,
    [int]$Port = 22,
    [string]$WorkspaceRoot = "E:\CTF\2026",
    [string]$ControllerPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHQ+sv570g0wEM6j0UJYnB/WT5TFLXcaJ1zypuUN2RpA sheathed-edge-controller",
    [switch]$AllowPasswordAuth,
    [string]$ReportPath = "$env:PUBLIC\Documents\sheathed-edge-ssh-report.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-Admin {
    $currentIdentity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($currentIdentity)
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw "Please run this script from an elevated PowerShell window (Run as Administrator)."
    }
}

function Ensure-Dir([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return
    }
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Ensure-OpenSSHServer {
    $capability = Get-WindowsCapability -Online | Where-Object Name -like 'OpenSSH.Server*' | Select-Object -First 1
    if ($null -eq $capability) {
        throw "OpenSSH.Server Windows Capability was not found on this machine."
    }
    if ($capability.State -ne 'Installed') {
        Add-WindowsCapability -Online -Name $capability.Name | Out-Null
    }
}

function Set-LineValue {
    param(
        [System.Collections.Generic.List[string]]$Lines,
        [string]$Key,
        [string]$Value
    )

    $pattern = "^\s*#?\s*$([regex]::Escape($Key))\b"
    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -match $pattern) {
            $Lines[$i] = "$Key $Value"
            return
        }
    }
    $Lines.Add("$Key $Value")
}

function Ensure-SshdConfig {
    param(
        [string]$ConfigPath,
        [string]$UserName,
        [int]$ListenPort,
        [bool]$EnablePasswordAuth
    )

    Ensure-Dir (Split-Path -Parent $ConfigPath)
    if (-not (Test-Path -LiteralPath $ConfigPath)) {
        New-Item -ItemType File -Path $ConfigPath -Force | Out-Null
    }

    $backupPath = "$ConfigPath.bak-sheathed-edge"
    Copy-Item -LiteralPath $ConfigPath -Destination $backupPath -Force

    $rawLines = Get-Content -LiteralPath $ConfigPath -ErrorAction SilentlyContinue
    $lines = [System.Collections.Generic.List[string]]::new()
    foreach ($line in $rawLines) {
        $lines.Add([string]$line)
    }

    Set-LineValue -Lines $lines -Key 'Port' -Value $ListenPort
    Set-LineValue -Lines $lines -Key 'PubkeyAuthentication' -Value 'yes'
    Set-LineValue -Lines $lines -Key 'PasswordAuthentication' -Value ($(if ($EnablePasswordAuth) { 'yes' } else { 'no' }))
    Set-LineValue -Lines $lines -Key 'PermitEmptyPasswords' -Value 'no'
    Set-LineValue -Lines $lines -Key 'TCPKeepAlive' -Value 'yes'
    Set-LineValue -Lines $lines -Key 'ClientAliveInterval' -Value '60'
    Set-LineValue -Lines $lines -Key 'ClientAliveCountMax' -Value '120'
    Set-LineValue -Lines $lines -Key 'AllowUsers' -Value $UserName

    Set-Content -LiteralPath $ConfigPath -Value $lines -Encoding ascii
}

function Ensure-AuthorizedKey {
    param(
        [string]$UserName,
        [string]$PublicKey
    )

    $userProfile = (Get-CimInstance Win32_UserProfile | Where-Object { $_.LocalPath -match "\\$([regex]::Escape($UserName))$" } | Sort-Object LastUseTime -Descending | Select-Object -First 1).LocalPath
    if ([string]::IsNullOrWhiteSpace($userProfile)) {
        $userProfile = Join-Path 'C:\Users' $UserName
    }

    $sshDir = Join-Path $userProfile '.ssh'
    $authorizedKeys = Join-Path $sshDir 'authorized_keys'
    Ensure-Dir $sshDir
    if (-not (Test-Path -LiteralPath $authorizedKeys)) {
        New-Item -ItemType File -Path $authorizedKeys -Force | Out-Null
    }

    $existing = Get-Content -LiteralPath $authorizedKeys -ErrorAction SilentlyContinue
    if (-not ($existing | Where-Object { $_.Trim() -eq $PublicKey.Trim() })) {
        Add-Content -LiteralPath $authorizedKeys -Value $PublicKey
    }

    & icacls $sshDir /inheritance:r | Out-Null
    & icacls $sshDir /grant:r "${UserName}:(OI)(CI)F" "SYSTEM:(OI)(CI)F" | Out-Null
    & icacls $authorizedKeys /inheritance:r | Out-Null
    & icacls $authorizedKeys /grant:r "${UserName}:F" "SYSTEM:F" | Out-Null

    return $authorizedKeys
}

function Ensure-FirewallRule {
    param(
        [int]$ListenPort
    )

    $ruleName = "Sheathed-Edge SSH $ListenPort"
    $existing = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if ($null -eq $existing) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Action Allow -Protocol TCP -LocalPort $ListenPort | Out-Null
    }
}

function Set-DefaultShellToPowerShell {
    $openSshRegPath = 'HKLM:\SOFTWARE\OpenSSH'
    if (-not (Test-Path -LiteralPath $openSshRegPath)) {
        New-Item -Path $openSshRegPath -Force | Out-Null
    }
    Set-ItemProperty -Path $openSshRegPath -Name DefaultShell -Value "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
}

function Restart-Sshd {
    Set-Service -Name sshd -StartupType Automatic
    if (Get-Service -Name ssh-agent -ErrorAction SilentlyContinue) {
        Set-Service -Name ssh-agent -StartupType Manual
    }
    Restart-Service -Name sshd -Force
}

function Get-HostFingerprints {
    $items = @()
    Get-ChildItem -LiteralPath 'C:\ProgramData\ssh' -Filter 'ssh_host_*_key.pub' -ErrorAction SilentlyContinue | ForEach-Object {
        $fingerprint = (& ssh-keygen -lf $_.FullName 2>$null)
        if (-not [string]::IsNullOrWhiteSpace($fingerprint)) {
            $items += [PSCustomObject]@{
                file = $_.Name
                fingerprint = $fingerprint.Trim()
            }
        }
    }
    return $items
}

function Get-IPSummary {
    $records = @()
    Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
        Where-Object {
            $_.IPAddress -notmatch '^169\.254\.' -and
            $_.IPAddress -ne '127.0.0.1'
        } |
        ForEach-Object {
            $adapter = Get-NetAdapter -InterfaceIndex $_.InterfaceIndex -ErrorAction SilentlyContinue
            $records += [PSCustomObject]@{
                interface = $adapter.Name
                ip = $_.IPAddress
                prefix = $_.PrefixLength
            }
        }
    return $records
}

function Invoke-LocalSshProbe {
    param(
        [string]$UserName,
        [int]$ListenPort
    )

    $result = [ordered]@{
        tcp_open = $false
        ssh_banner = ""
        loopback_note = ""
    }

    $tcp = Test-NetConnection -ComputerName 127.0.0.1 -Port $ListenPort -WarningAction SilentlyContinue
    $result.tcp_open = [bool]$tcp.TcpTestSucceeded

    try {
        $client = [System.Net.Sockets.TcpClient]::new()
        $client.ReceiveTimeout = 3000
        $client.SendTimeout = 3000
        $client.Connect('127.0.0.1', $ListenPort)
        $stream = $client.GetStream()
        $buffer = New-Object byte[] 512
        $read = $stream.Read($buffer, 0, $buffer.Length)
        if ($read -gt 0) {
            $result.ssh_banner = [System.Text.Encoding]::ASCII.GetString($buffer, 0, $read).Trim()
        }
        $stream.Dispose()
        $client.Dispose()
    } catch {
        $result.loopback_note = $_.Exception.Message
    }

    return [PSCustomObject]$result
}

Assert-Admin

Write-Host "[1/8] Install/check OpenSSH Server..."
Ensure-OpenSSHServer

Write-Host "[2/8] Configure sshd..."
$sshdConfigPath = 'C:\ProgramData\ssh\sshd_config'
Ensure-SshdConfig -ConfigPath $sshdConfigPath -UserName $Username -ListenPort $Port -EnablePasswordAuth:$AllowPasswordAuth.IsPresent

Write-Host "[3/8] Write authorized_keys..."
$authorizedKeysPath = Ensure-AuthorizedKey -UserName $Username -PublicKey $ControllerPublicKey

Write-Host "[4/8] Set default shell..."
Set-DefaultShellToPowerShell

Write-Host "[5/8] Open firewall rule..."
Ensure-FirewallRule -ListenPort $Port

Write-Host "[6/8] Start service and enable auto-start..."
Restart-Sshd

Write-Host "[7/8] Generate report..."
Ensure-Dir (Split-Path -Parent $ReportPath)

$service = Get-Service sshd
$listener = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
$probe = Invoke-LocalSshProbe -UserName $Username -ListenPort $Port
$fingerprints = Get-HostFingerprints
$ips = Get-IPSummary

$report = [ordered]@{
    ok = ($service.Status -eq 'Running' -and $probe.tcp_open)
    timestamp = (Get-Date).ToString('s')
    computer_name = $env:COMPUTERNAME
    username = $Username
    workspace_root = $WorkspaceRoot
    ssh = [ordered]@{
        service_status = $service.Status.ToString()
        startup_type = (Get-CimInstance Win32_Service -Filter "Name='sshd'").StartMode
        port = $Port
        listener = if ($listener) { "$($listener.LocalAddress):$($listener.LocalPort)" } else { "" }
        banner = $probe.ssh_banner
        tcp_open = $probe.tcp_open
        loopback_note = $probe.loopback_note
        default_shell = (Get-ItemProperty -Path 'HKLM:\SOFTWARE\OpenSSH' -Name DefaultShell -ErrorAction SilentlyContinue).DefaultShell
        authorized_keys = $authorizedKeysPath
        fingerprints = $fingerprints
    }
    network = [ordered]@{
        ipv4 = $ips
    }
    next_step = "Send this JSON file and console output back to the controller. Then verify from the controller with: ssh $Username@<ip> -p $Port"
}

$report | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $ReportPath -Encoding utf8

Write-Host "[8/8] Done."
Write-Host ""
Write-Host "===== SSH setup result =====" -ForegroundColor Cyan
$report | ConvertTo-Json -Depth 6
Write-Host ""
Write-Host "Report saved to: $ReportPath" -ForegroundColor Green
