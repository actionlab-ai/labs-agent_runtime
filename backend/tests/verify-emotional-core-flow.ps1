<#
.SYNOPSIS
Verify the transparent project -> tool_search -> skill -> provider-backed document flow.

.EXAMPLE
powershell -ExecutionPolicy Bypass -File .\tests\verify-emotional-core-flow.ps1 -Model deepseek-flash

.EXAMPLE
# If /v1/settings/default-model is already configured:
powershell -ExecutionPolicy Bypass -File .\tests\verify-emotional-core-flow.ps1
#>

param(
    [string]$BaseUrl = "http://127.0.0.1:8080",
    [string]$ProjectId = ("emotional-core-flow-" + (Get-Date -Format "yyyyMMddHHmmss")),
    [string]$Model = "",
    [string]$DocumentKind = "novel_core",
    [switch]$KeepProject,
    [int]$TimeoutSec = 180
)

$ErrorActionPreference = "Stop"

$BackendRoot = Split-Path -Parent $PSScriptRoot
$LogFile = Join-Path $BackendRoot "logs\novelrt-latest.log"

function Write-Step([string]$Message) {
    Write-Host ""
    Write-Host ">>> $Message" -ForegroundColor Cyan
}

function Write-Ok([string]$Message) {
    Write-Host "OK  $Message" -ForegroundColor Green
}

function Invoke-Api {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Path,
        [object]$Body = $null,
        [int]$Timeout = 30
    )

    $uri = "$BaseUrl$Path"
    $headers = @{ "X-Request-ID" = [Guid]::NewGuid().ToString("N") }
    $requestParams = @{
        Uri             = $uri
        Method          = $Method
        Headers         = $headers
        ContentType     = "application/json; charset=utf-8"
        UseBasicParsing = $true
        TimeoutSec      = $Timeout
    }
    if ($null -ne $Body) {
        $requestParams.Body = ($Body | ConvertTo-Json -Depth 20)
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
                Write-Host "Response body: $($reader.ReadToEnd())" -ForegroundColor Red
            } catch {
                Write-Host "Failed to read error response body." -ForegroundColor Yellow
            }
        }
        throw
    }
}

function Read-JsonFile([string]$Path) {
    if (-not (Test-Path $Path)) {
        throw "Expected file not found: $Path"
    }
    $content = [System.IO.File]::ReadAllText($Path, [System.Text.Encoding]::UTF8)
    return $content | ConvertFrom-Json
}

function Get-ToolNames($Assembly) {
    $names = @()
    foreach ($tool in @($Assembly.tools)) {
        if ($tool.function.name) {
            $names += [string]$tool.function.name
        }
    }
    return $names
}

function Get-ResponseToolNames($Analysis) {
    $names = @()
    foreach ($name in @($Analysis.tool_call_names)) {
        if ($name) {
            $names += [string]$name
        }
    }
    return $names
}

function Get-ToolMessagePayloads($Assembly) {
    $payloads = @()
    foreach ($msg in @($Assembly.messages)) {
        if ($msg.role -eq "tool" -and $msg.content) {
            try {
                $payloads += ($msg.content | ConvertFrom-Json)
            } catch {
                # Keep script useful even if a tool payload is not JSON.
            }
        }
    }
    return $payloads
}

function Find-FirstFile([string]$Dir, [string]$Filter) {
    $file = Get-ChildItem -Path $Dir -Filter $Filter -File -ErrorAction SilentlyContinue |
        Sort-Object Name |
        Select-Object -First 1
    if ($file) {
        return $file.FullName
    }
    return $null
}

function Assert-ContainsTool([string[]]$Names, [string]$Expected, [string]$Stage) {
    if ($Names -notcontains $Expected) {
        throw "$Stage did not include expected tool '$Expected'. Actual tools: $($Names -join ', ')"
    }
}

function Assert-AnyStartsWith([string[]]$Names, [string]$Prefix, [string]$Stage) {
    foreach ($name in $Names) {
        if ($name.StartsWith($Prefix)) {
            return $name
        }
    }
    throw "$Stage did not include a tool starting with '$Prefix'. Actual tools: $($Names -join ', ')"
}

function Try-DeleteProject([string]$ID) {
    try {
        [void](Invoke-Api -Method "DELETE" -Path "/v1/projects/$ID" -Timeout 10)
    } catch {
        # Ignore cleanup failures.
    }
}

function Assert-ModelAvailable([string]$ModelID) {
    if ($ModelID.Trim() -eq "") {
        return
    }
    $modelsResp = Invoke-Api -Method "GET" -Path "/v1/models" -Timeout 10
    $available = @()
    foreach ($item in @($modelsResp.Body.models)) {
        if ($item.id) {
            $available += [string]$item.id
        }
    }
    if ($available -notcontains $ModelID) {
        throw "Model '$ModelID' was not found in /v1/models. Available model ids: $($available -join ', '). Use one of those ids or set /v1/settings/default-model."
    }
}

Write-Host "===== verify emotional core skill flow =====" -ForegroundColor Yellow
Write-Host "Base URL      : $BaseUrl"
Write-Host "Project ID    : $ProjectId"
Write-Host "Model         : $(if ($Model) { $Model } else { '<database default model>' })"
Write-Host "Document kind : $DocumentKind"
Write-Host "Backend root  : $BackendRoot"
Write-Host "Log file      : $LogFile"

Write-Step "0) Health check"
[void](Invoke-Api -Method "GET" -Path "/healthz" -Timeout 10)
Write-Ok "HTTP service is reachable."
Assert-ModelAvailable -ModelID $Model
if ($Model.Trim() -ne "") {
    Write-Ok "Model profile exists: $Model"
}

if (-not $KeepProject) {
    Try-DeleteProject -ID $ProjectId
}

Write-Step "1) Create a filesystem-backed project"
$createResp = Invoke-Api -Method "POST" -Path "/v1/projects" -Body @{
    id               = $ProjectId
    name             = "emotional-core-transparent-flow"
    description      = "verify project -> tool_search -> skill -> provider-backed document"
    storage_provider = "filesystem"
    storage_prefix   = $ProjectId
} -Timeout 20

if ($createResp.Body.project.id -ne $ProjectId) {
    throw "Unexpected project response: $($createResp.RawBody)"
}
Write-Ok "Project created: $($createResp.Body.project.id)"

Write-Step "2) Run runtime and force the first business document to be emotional core"
$inputText = @(
    "Start a new Chinese webnovel project, but do not begin with worldbuilding."
    "First use tool_search and activate select:novel-emotional-core."
    "Then call that skill and create project document kind $DocumentKind."
    ""
    "Genre: urban supernatural realistic-pressure power fantasy."
    "Target reader: middle-aged men crushed by life, family, and workplace pressure."
    "Protagonist: 38-year-old ordinary salesman, long ignored and disrespected."
    "Deep emotional need: to be seen again and regain respect."
    "Pressure source: KPI, debt, family duty, and humiliation by a younger manager."
    "Payoff: not simple sudden wealth, but gradually regaining choice, agency, and dignity."
    ""
    "Write the final project document in Chinese."
    "Inside the skill executor, you must use WriteProjectDocument. The kind must be $DocumentKind."
) -join [Environment]::NewLine

$runBody = @{
    project = $ProjectId
    input   = $inputText
    debug   = $true
}
if ($Model.Trim() -ne "") {
    $runBody.model = $Model.Trim()
}

$runResp = Invoke-Api -Method "POST" -Path "/v1/runs" -Body $runBody -Timeout $TimeoutSec
$runDir = [string]$runResp.Body.run_dir
if (-not $runDir -or -not (Test-Path $runDir)) {
    throw "Run completed but run_dir is missing or not found. Response: $($runResp.RawBody)"
}
Write-Ok "Run completed. run_id=$($runResp.Body.run_id)"
Write-Host "Run dir: $runDir" -ForegroundColor Gray

Write-Step "3) Inspect router round 01: only tool_search should be exposed and called"
$router01AssemblyPath = Join-Path $runDir "router\round-01-assembly.json"
$router01AnalysisPath = Join-Path $runDir "router\round-01-response-analysis.json"
$router01 = Read-JsonFile $router01AssemblyPath
$router01Analysis = Read-JsonFile $router01AnalysisPath

$router01Tools = Get-ToolNames $router01
$router01Called = Get-ResponseToolNames $router01Analysis
Write-Host "Router round 01 exposed tools: $($router01Tools -join ', ')"
Write-Host "Router round 01 called tools : $($router01Called -join ', ')"
Assert-ContainsTool -Names $router01Tools -Expected "tool_search" -Stage "router round 01 exposed tools"
Assert-ContainsTool -Names $router01Called -Expected "tool_search" -Stage "router round 01 model response"
Write-Ok "tool_search gate is visible and was used."

Write-Step "4) Inspect router round 02: tool_search result should retain novel-emotional-core"
$router02AssemblyPath = Join-Path $runDir "router\round-02-assembly.json"
$router02AnalysisPath = Join-Path $runDir "router\round-02-response-analysis.json"
$router02 = Read-JsonFile $router02AssemblyPath
$router02Analysis = Read-JsonFile $router02AnalysisPath

$toolPayloads = Get-ToolMessagePayloads $router02
$searchPayload = $toolPayloads | Where-Object { $_.type -eq "tool_search_result" } | Select-Object -First 1
if (-not $searchPayload) {
    throw "Could not find tool_search_result payload in router round 02 messages. Check $router02AssemblyPath"
}

$hitIDs = @()
foreach ($hit in @($searchPayload.hits)) {
    if ($hit.id) {
        $hitIDs += [string]$hit.id
    }
}
$retainedIDs = @()
foreach ($item in @($searchPayload.activation.retained_skills)) {
    if ($item.skill_id) {
        $retainedIDs += [string]$item.skill_id
    }
}

Write-Host "tool_search hits             : $($hitIDs -join ', ')"
Write-Host "retained discovered skills   : $($retainedIDs -join ', ')"
if ($retainedIDs -notcontains "novel-emotional-core") {
    throw "tool_search did not retain novel-emotional-core. Hits: $($hitIDs -join ', ')"
}

$router02Tools = Get-ToolNames $router02
$router02Called = Get-ResponseToolNames $router02Analysis
Write-Host "Router round 02 exposed tools: $($router02Tools -join ', ')"
Write-Host "Router round 02 called tools : $($router02Called -join ', ')"
$skillToolName = Assert-AnyStartsWith -Names $router02Called -Prefix "skill_exec_novel_emotional_core_" -Stage "router round 02 model response"
Write-Ok "novel-emotional-core was activated and called through $skillToolName."

Write-Step "5) Inspect skill executor: provider-backed project document tools should be available and used"
$skillDir = Join-Path $runDir "skill-calls\novel-emotional-core"
$skillAssemblyFiles = Get-ChildItem -Path $skillDir -Filter "round-*-assembly.json" -File | Sort-Object Name
$skillAnalysisFiles = Get-ChildItem -Path $skillDir -Filter "round-*-response-analysis.json" -File | Sort-Object Name
if ($skillAssemblyFiles.Count -eq 0 -or $skillAnalysisFiles.Count -eq 0) {
    throw "Skill executor artifacts are missing under $skillDir"
}

$allSkillLocalTools = @()
foreach ($file in $skillAssemblyFiles) {
    $assembly = Read-JsonFile $file.FullName
    foreach ($tool in @($assembly.local_tools)) {
        if ($tool.name) {
            $allSkillLocalTools += [string]$tool.name
        }
    }
}

$allSkillCalled = @()
foreach ($file in $skillAnalysisFiles) {
    $analysis = Read-JsonFile $file.FullName
    $roundCalled = Get-ResponseToolNames $analysis
    if ($roundCalled.Count -gt 0) {
        Write-Host "$($file.BaseName) called tools : $($roundCalled -join ', ')"
    }
    $allSkillCalled += $roundCalled
}

$allSkillLocalTools = $allSkillLocalTools | Select-Object -Unique
$allSkillCalled = $allSkillCalled | Select-Object -Unique
Write-Host "Skill local tools            : $($allSkillLocalTools -join ', ')"
Write-Host "Skill called tools overall   : $($allSkillCalled -join ', ')"
Assert-ContainsTool -Names $allSkillLocalTools -Expected "WriteProjectDocument" -Stage "skill local tools"
Assert-ContainsTool -Names $allSkillCalled -Expected "WriteProjectDocument" -Stage "skill executor model responses"
Write-Ok "Skill used provider-backed WriteProjectDocument instead of writing a path directly."

Write-Step "6) Read project documents through HTTP and verify $DocumentKind exists"
$docsResp = Invoke-Api -Method "GET" -Path "/v1/projects/$ProjectId/documents" -Timeout 20
$doc = $docsResp.Body.documents | Where-Object { $_.kind -eq $DocumentKind } | Select-Object -First 1
if (-not $doc) {
    throw "Project document '$DocumentKind' was not found. Payload: $($docsResp.RawBody)"
}
Write-Ok "Project document exists in API response: kind=$($doc.kind), title=$($doc.title)"

Write-Step "7) Print the transparent artifact paths"
$compiledPromptPath = Join-Path $skillDir "compiled-prompt.md"
$skillAssemblyMd = Join-Path $skillDir "round-01-assembly.md"
$router01Md = Join-Path $runDir "router\round-01-assembly.md"
$router02Md = Join-Path $runDir "router\round-02-assembly.md"

Write-Host "Router round 01 assembly : $router01Md"
Write-Host "Router round 02 assembly : $router02Md"
Write-Host "Skill compiled prompt    : $compiledPromptPath"
Write-Host "Skill round 01 assembly  : $skillAssemblyMd"
Write-Host "Project FS projection    : $(Join-Path $BackendRoot "..\projects\$ProjectId\documents\$DocumentKind.md")"

Write-Host ""
Write-Host "===== flow summary =====" -ForegroundColor Yellow
Write-Host "1. POST /v1/projects created the project row and filesystem projection."
Write-Host "2. POST /v1/runs round 01 exposed only tool_search."
Write-Host "3. The model called tool_search; the runtime retained novel-emotional-core."
Write-Host "4. Router round 02 exposed the activated skill tool and the model called it."
Write-Host "5. The skill executor exposed WriteProjectDocument and the model called it."
Write-Host "6. WriteProjectDocument went through ProjectDocumentProvider, so PG/Redis/filesystem sync stays outside the business skill."

Write-Host ""
Write-Host "The generated project is kept for inspection." -ForegroundColor DarkYellow
Write-Host "Delete later with: Invoke-RestMethod -Method DELETE '$BaseUrl/v1/projects/$ProjectId'"
