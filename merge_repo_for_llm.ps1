param(
    [string]$Out = "repo_merged.md",
    [int]$MaxFileSize = 300000
)

$ErrorActionPreference = "Stop"

if (Test-Path $Out) {
    Remove-Item $Out -Force
}

function Add-Line {
    param([string]$Text = "")
    Add-Content -Path $Out -Value $Text -Encoding UTF8
}
function Should-Skip {
    param([string]$File)

    $path = $File.Replace("\", "/")

    # 跳过目录
    $skipDirs = @(
        ".git/",
        "vendor/",
        "node_modules/",
        "dist/",
        "build/",
        "bin/",
        "target/",
        "tmp/",
        ".idea/",
        ".vscode/",

        # Go / 编译 / 模块缓存
        "gocache/",
        ".gocache/",
        "gomodcache/",
        ".gomodcache/",
        "go-build/",
        ".go-build/",
        ".cache/",
        "cache/",
        "pkg/mod/",
        "pkg/sumdb/",
        "pkg/mod/cache/"
    )

    foreach ($dir in $skipDirs) {
        if ($path.StartsWith($dir) -or $path.Contains("/$dir")) {
            return $true
        }
    }

    # 跳过常见缓存目录名
    $parts = $path -split "/"
    foreach ($part in $parts) {
        if ($part -in @(
            "gocache",
            ".gocache",
            "gomodcache",
            ".gomodcache",
            "go-build",
            ".go-build",
            ".cache",
            "__pycache__"
        )) {
            return $true
        }
    }

    # 跳过二进制、图片、压缩包、日志、证书、私钥
    $skipPatterns = @(
        "\.png$",
        "\.jpg$",
        "\.jpeg$",
        "\.gif$",
        "\.webp$",
        "\.ico$",
        "\.svg$",

        "\.zip$",
        "\.tar$",
        "\.gz$",
        "\.tgz$",
        "\.rar$",
        "\.7z$",
        "\.xz$",
        "\.bz2$",

        "\.exe$",
        "\.dll$",
        "\.so$",
        "\.dylib$",
        "\.a$",
        "\.o$",
        "\.class$",
        "\.jar$",
        "\.war$",

        "\.log$",
        "\.pid$",
        "\.lock$",

        "\.pem$",
        "\.key$",
        "\.crt$",
        "\.p12$",
        "\.jks$"
    )

    foreach ($pattern in $skipPatterns) {
        if ($path -match $pattern) {
            return $true
        }
    }

    # 跳过真实 .env，但保留 .env.example / .env.sample
    $name = Split-Path $path -Leaf

    if ($name -eq ".env") {
        return $true
    }

    if ($name -match "^\.env\.(local|dev|prod|production|test)$") {
        return $true
    }

    if ($name -match ".*\.env$") {
        return $true
    }

    return $false
}

function Is-BinaryFile {
    param([string]$File)

    try {
        $bytes = [System.IO.File]::ReadAllBytes($File)
        $checkLength = [Math]::Min($bytes.Length, 8000)

        for ($i = 0; $i -lt $checkLength; $i++) {
            if ($bytes[$i] -eq 0) {
                return $true
            }
        }

        return $false
    }
    catch {
        return $true
    }
}

function Get-Lang {
    param([string]$File)

    $name = Split-Path $File -Leaf
    $ext = [System.IO.Path]::GetExtension($File).ToLower()

    switch ($ext) {
        ".go" { return "go" }
        ".md" { return "markdown" }
        ".yaml" { return "yaml" }
        ".yml" { return "yaml" }
        ".json" { return "json" }
        ".toml" { return "toml" }
        ".ini" { return "ini" }
        ".conf" { return "ini" }
        ".sh" { return "bash" }
        ".bash" { return "bash" }
        ".ps1" { return "powershell" }
        ".sql" { return "sql" }
        ".proto" { return "protobuf" }
        ".mod" { return "go" }
        ".sum" { return "" }
        ".xml" { return "xml" }
        ".html" { return "html" }
        ".css" { return "css" }
        ".js" { return "javascript" }
        ".ts" { return "typescript" }
        ".env.example" { return "bash" }
        ".env.sample" { return "bash" }
        default {
            if ($name -eq "Dockerfile") { return "dockerfile" }
            if ($name -eq "Makefile") { return "makefile" }
            return ""
        }
    }
}

function Get-RepoFiles {
    $gitExists = Get-Command git -ErrorAction SilentlyContinue

    if ($gitExists) {
        try {
            git rev-parse --is-inside-work-tree 2>$null | Out-Null

            if ($LASTEXITCODE -eq 0) {
                $files = git ls-files -co --exclude-standard
                return $files
            }
        }
        catch {
            # fallback to Get-ChildItem
        }
    }

    return Get-ChildItem -Recurse -File |
        ForEach-Object {
            Resolve-Path -Relative $_.FullName
        } |
        ForEach-Object {
            $_.TrimStart(".\").TrimStart("./")
        }
}

# Header
Add-Line "# Repository Merged For LLM"
Add-Line ""
Add-Line "Generated at: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Add-Line ""
Add-Line "Purpose: merge source code, documents, and configuration files into one Markdown file for LLM analysis."
Add-Line ""
Add-Line "---"
Add-Line ""

# Repository Tree
Add-Line "## Repository Tree"
Add-Line ""
Add-Line '```text'

$allFiles = Get-RepoFiles | Sort-Object -Unique

foreach ($file in $allFiles) {
    if (-not (Should-Skip $file)) {
        Add-Line $file
    }
}

Add-Line '```'
Add-Line ""
Add-Line "---"
Add-Line ""

# Files
Add-Line "## Files"
Add-Line ""

foreach ($file in $allFiles) {
    if (-not (Test-Path $file)) {
        continue
    }

    if (Should-Skip $file) {
        continue
    }

    $item = Get-Item $file

    if ($item.Length -gt $MaxFileSize) {
        Add-Line ""
        Add-Line "### File: $file"
        Add-Line ""
        Add-Line "> Skipped: file too large ($($item.Length) bytes, limit $MaxFileSize bytes)."
        Add-Line ""
        continue
    }

    if (Is-BinaryFile $file) {
        continue
    }

    $lang = Get-Lang $file

    Add-Line ""
    Add-Line "### File: $file"
    Add-Line ""
    Add-Line "````$lang"

    try {
        Get-Content -Path $file -Raw -Encoding UTF8 | Add-Content -Path $Out -Encoding UTF8
    }
    catch {
        try {
            Get-Content -Path $file -Raw | Add-Content -Path $Out -Encoding UTF8
        }
        catch {
            Add-Line "> Failed to read file: $file"
        }
    }

    Add-Line ""
    Add-Line "````"
}

Write-Host "OK: merged repository into $Out" -ForegroundColor Green
Write-Host "Example: .\merge_repo_for_llm.ps1 -Out repo_merged.md -MaxFileSize 500000"