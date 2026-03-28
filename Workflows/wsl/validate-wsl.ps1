param(
    [string]$DistroName = "Ubuntu-24.04",
    [string]$ProjectRootWsl = "/mnt/d/AI/Local"
)

$ErrorActionPreference = "Stop"

& wsl.exe -d $DistroName -- bash /mnt/d/AI/Local/Workflows/wsl/validate-wsl.sh $ProjectRootWsl
if ($LASTEXITCODE -ne 0) {
    throw "validate-wsl.sh 执行失败，退出码: $LASTEXITCODE"
}
