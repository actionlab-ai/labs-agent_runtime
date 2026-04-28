$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Verify project create/update/delete cache behavior without redis-cli.
# It validates Redis-related behavior through HTTP + server logs.
# -----------------------------------------------------------------------------

$BaseUrl   = "http://127.0.0.1:8080"
$ProjectId = "test-project-cache"
$LogFile   = Join-Path (Split-Path -Parent $PSScriptRoot) "logs\novelrt-latest.log"

function Write-Step([string]$Message) {
    Write-Host ""
    Write-Host ">>> $Message" -ForegroundColor Cyan
}

function Invoke-Api {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Path,
        [object]$Body = $null
    )

    $uri = "$BaseUrl$Path"
    $headers = @{ "X-Request-ID" = [Guid]::NewGuid().ToString("N") }
    $requestParams = @{
        Uri             = $uri
        Method          = $Method
        Headers         = $headers
        ContentType     = "application/json"
        UseBasicParsing = $true
    }
    if ($null -ne $Body) {
        $requestParams.Body = ($Body | ConvertTo-Json -Depth 10)
    }

    try {
        $response = Invoke-WebRequest @requestParams
        $requestID = $response.Headers["X-Request-ID"]
        if (-not $requestID) {
            $requestID = $headers["X-Request-ID"]
        }
        $json = $null
        if ($response.Content) {
            $json = $response.Content | ConvertFrom-Json
        }
        return [PSCustomObject]@{
            RequestID = $requestID
            Status    = [int]$response.StatusCode
            Body      = $json
            RawBody   = $response.Content
        }
    } catch {
        Write-Host "HTTP request failed: $Method $uri" -ForegroundColor Red
        if ($_.Exception.Response) {
            try {
                $stream = $_.Exception.Response.GetResponseStream()
                $reader = New-Object System.IO.StreamReader($stream)
                $errBody = $reader.ReadToEnd()
                Write-Host "Response body: $errBody" -ForegroundColor Red
            } catch {
                Write-Host "Failed to read error response body." -ForegroundColor Yellow
            }
        }
        throw
    }
}

function Assert-LogLine {
    param(
        [Parameter(Mandatory = $true)][string]$RequestID,
        [Parameter(Mandatory = $true)][string]$MustContain,
        [int]$TimeoutSec = 12
    )

    if (-not (Test-Path $LogFile)) {
        throw "Log file not found: $LogFile"
    }

    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        $tail = Get-Content $LogFile -Tail 1200 -ErrorAction SilentlyContinue
        foreach ($line in $tail) {
            if ($line -like "*$RequestID*" -and $line -like "*$MustContain*") {
                return $line
            }
        }
        Start-Sleep -Milliseconds 300
    }

    throw "Expected log not found. request_id=$RequestID mustContain=$MustContain"
}

function Try-DeleteProject {
    param([string]$ID)
    try {
        [void](Invoke-Api -Method "DELETE" -Path "/v1/projects/$ID")
    } catch {
        # Ignore cleanup failure (project may not exist).
    }
}

Write-Host "===== verify project cache flow (no redis-cli) =====" -ForegroundColor Yellow
Write-Host "Base URL: $BaseUrl"
Write-Host "Project ID: $ProjectId"
Write-Host "Log file: $LogFile"

Write-Step "0) Health check + cleanup"
[void](Invoke-Api -Method "GET" -Path "/healthz")
Try-DeleteProject -ID $ProjectId
Start-Sleep -Milliseconds 500

Write-Step "1) Create project (expect PG create + cache set)"
$createResp = Invoke-Api -Method "POST" -Path "/v1/projects" -Body @{
    id               = $ProjectId
    name             = "cache-verify-project"
    description      = "verify pg+redis cache sync without redis-cli"
    storage_provider = "filesystem"
    storage_prefix   = $ProjectId
}
if ($createResp.Status -ne 200 -or $createResp.Body.project.id -ne $ProjectId) {
    throw "Create project failed or unexpected payload."
}
[void](Assert-LogLine -RequestID $createResp.RequestID -MustContain "project.pg.create")
[void](Assert-LogLine -RequestID $createResp.RequestID -MustContain '"op": "project.cache.set"')
[void](Assert-LogLine -RequestID $createResp.RequestID -MustContain "cache.sync.done")
Write-Host "Create verified." -ForegroundColor Green

Write-Step "2) Get project (expect cache hit)"
$getResp = Invoke-Api -Method "GET" -Path "/v1/projects/$ProjectId"
if ($getResp.Status -ne 200 -or $getResp.Body.project.id -ne $ProjectId) {
    throw "Get project failed or unexpected payload."
}
[void](Assert-LogLine -RequestID $getResp.RequestID -MustContain "project.cache.hit")
Write-Host "Cache hit verified." -ForegroundColor Green

Write-Step "3) Update project (expect PG update + cache set)"
$updateResp = Invoke-Api -Method "PATCH" -Path "/v1/projects/$ProjectId" -Body @{
    name        = "cache-verify-project-updated"
    description = "updated by verify script"
}
if ($updateResp.Status -ne 200 -or $updateResp.Body.project.name -ne "cache-verify-project-updated") {
    throw "Update project failed or unexpected payload."
}
[void](Assert-LogLine -RequestID $updateResp.RequestID -MustContain "project.pg.update")
[void](Assert-LogLine -RequestID $updateResp.RequestID -MustContain '"op": "project.cache.set"')
[void](Assert-LogLine -RequestID $updateResp.RequestID -MustContain "cache.sync.done")
Write-Host "Update verified." -ForegroundColor Green

Write-Step "4) Delete project (expect PG delete + cache delete)"
$deleteResp = Invoke-Api -Method "DELETE" -Path "/v1/projects/$ProjectId"
if ($deleteResp.Status -ne 200) {
    throw "Delete project failed."
}
[void](Assert-LogLine -RequestID $deleteResp.RequestID -MustContain "project.pg.delete")
[void](Assert-LogLine -RequestID $deleteResp.RequestID -MustContain '"op": "project.cache.delete"')
[void](Assert-LogLine -RequestID $deleteResp.RequestID -MustContain "cache.sync.done")
Write-Host "Delete verified." -ForegroundColor Green

Write-Step "5) Get deleted project (expect not found)"
$notFound = $false
try {
    [void](Invoke-Api -Method "GET" -Path "/v1/projects/$ProjectId")
} catch {
    $notFound = $true
}
if (-not $notFound) {
    throw "Expected project not found after delete."
}
Write-Host "Not-found after delete verified." -ForegroundColor Green

Write-Host ""
Write-Host "===== verification completed =====" -ForegroundColor Yellow
