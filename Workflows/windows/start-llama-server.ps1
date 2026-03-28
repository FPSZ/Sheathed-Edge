param(
    [string]$ConfigPath = "D:\AI\Local\Agent\llama-server.config.json",
    [switch]$Detached
)

$ErrorActionPreference = "Stop"

function Read-JsonFile {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Config file not found: $Path"
    }

    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Resolve-LlamaBinary {
    param($Config)

    if ($env:LLAMA_SERVER_BIN -and (Test-Path -LiteralPath $env:LLAMA_SERVER_BIN)) {
        return $env:LLAMA_SERVER_BIN
    }

    if ($Config.binary_path -and (Test-Path -LiteralPath $Config.binary_path)) {
        return $Config.binary_path
    }

    foreach ($candidate in $Config.binary_candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate)) {
            return $candidate
        }
        if ($candidate -and (Test-Path -LiteralPath (Split-Path -Parent $candidate))) {
            $found = Get-ChildItem -LiteralPath (Split-Path -Parent $candidate) -Recurse -Filter llama-server.exe -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($found) {
                return $found.FullName
            }
        }
    }

    $cmd = Get-Command llama-server.exe -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    throw "Unable to find llama-server.exe. Set LLAMA_SERVER_BIN or update Agent\\llama-server.config.json."
}

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Test-PortBusy {
    param([string]$BindAddress, [int]$Port)
    $conn = Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue | Where-Object { $_.LocalAddress -eq $BindAddress -and $_.LocalPort -eq $Port }
    return [bool]$conn
}

$config = Read-JsonFile -Path $ConfigPath
$binary = Resolve-LlamaBinary -Config $config
$modelPath = $config.model_path
$listenHost = $config.listen_host
$listenPort = [int]$config.listen_port
$logDir = $config.log_dir

if (-not (Test-Path -LiteralPath $modelPath)) {
    throw "Model file not found: $modelPath"
}

Ensure-Directory -Path $logDir

if (Test-PortBusy -BindAddress $listenHost -Port $listenPort) {
    throw "Port already in use: ${listenHost}:$listenPort"
}

$args = @(
    "--model", $modelPath,
    "--host", $listenHost,
    "--port", "$listenPort",
    "--ctx-size", "$($config.ctx_size)",
    "--n-gpu-layers", "$($config.n_gpu_layers)",
    "--threads", "$($config.threads)",
    "--parallel", "$($config.parallel)"
)

if ($config.flash_attn) {
    $args += @("--flash-attn", "on")
}

if ($config.cont_batching) {
    $args += "--cont-batching"
}

$stdoutLog = Join-Path $logDir "stdout.log"
$stderrLog = Join-Path $logDir "stderr.log"

Write-Host "Using llama-server binary: $binary"
Write-Host "Using model: $modelPath"
Write-Host "Listening on http://$listenHost`:$listenPort"

if ($Detached) {
    Start-Process -FilePath $binary -ArgumentList $args -WorkingDirectory (Split-Path -Parent $binary) -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog | Out-Null
    Write-Host "llama-server started in detached mode"
} else {
    & $binary @args
}
