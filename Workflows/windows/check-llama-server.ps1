param(
    [string]$BaseUrl = "http://127.0.0.1:8080"
)

$ErrorActionPreference = "Stop"

Write-Host "Checking $BaseUrl/health"
Invoke-RestMethod -Uri "$BaseUrl/health" -Method Get | Out-Host

Write-Host ""
Write-Host "Checking $BaseUrl/v1/models"
$models = Invoke-RestMethod -Uri "$BaseUrl/v1/models" -Method Get
$models | ConvertTo-Json -Depth 8

$modelId = $models.data[0].id
if (-not $modelId) {
    throw "No model id returned from $BaseUrl/v1/models"
}

$body = @{
    model = $modelId
    messages = @(
        @{
            role = "user"
            content = "ping"
        }
    )
    stream = $false
} | ConvertTo-Json -Depth 8

Write-Host ""
Write-Host "Checking $BaseUrl/v1/chat/completions"
Invoke-RestMethod -Uri "$BaseUrl/v1/chat/completions" -Method Post -ContentType "application/json" -Body $body | ConvertTo-Json -Depth 8
