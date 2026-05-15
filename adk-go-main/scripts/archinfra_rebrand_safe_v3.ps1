param(
  [switch]$FixMissingHeaders,
  [string]$Owner = "Archinfra / yuanyp8",
  [int]$GoogleStartYear = 2025,
  [int]$CustomStartYear = 2026
)

$ErrorActionPreference = "Stop"

$Utf8NoBom = New-Object System.Text.UTF8Encoding($false)

function Read-Utf8([string]$Path) {
  return [System.IO.File]::ReadAllText($Path, [System.Text.Encoding]::UTF8)
}

function Write-Utf8([string]$Path, [string]$Content) {
  [System.IO.File]::WriteAllText($Path, $Content, $Utf8NoBom)
}

function Is-IgnoredPath([string]$FullName) {
  $normalized = $FullName.Replace("/", "\")
  return (
    $normalized.Contains("\.git\") -or
    $normalized.Contains("\vendor\") -or
    $normalized.Contains("\internal\jsonschema\") -or
    $normalized.Contains("\internal\util\") -or
    $normalized.Contains("\internal\httprr\")
  )
}

function Has-AcceptedHeader([string]$Content) {
  if ($Content.StartsWith("// Copyright 2025 Google LLC")) { return $true }
  if ($Content.StartsWith("// Copyright 2026 Archinfra / yuanyp8")) { return $true }
  if ($Content.StartsWith("// Copyright 2025 Archinfra / yuanyp8")) { return $true }
  return $false
}

$Root = (Get-Location).Path
$StyleTest = Join-Path $Root "internal\style_test.go"
$Golangci = Join-Path $Root ".golangci.yml"
$ModelLLM = Join-Path $Root "model\llm.go"
$Notice = Join-Path $Root "NOTICE-ARCHINFRA.md"

if (!(Test-Path $StyleTest)) {
  throw "internal\style_test.go not found. Run this script from the module root."
}

$HeaderTestContent = @"
// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const archinfraCopyrightHeaderTmpl = ` + "`" + `// Copyright %d __OWNER__

//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
` + "`" + `

const googleCopyrightHeaderTmpl = ` + "`" + `// Copyright %d Google LLC

//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
` + "`" + `

const googleStartYear = __GOOGLE_START_YEAR__
const archinfraStartYear = __CUSTOM_START_YEAR__

var fixError = flag.Bool("fix", false, "fix detected problems, for example add missing copyright headers")

func TestCopyrightHeader(t *testing.T) {
	// Start test from the parent directory, root of the module.
	t.Chdir("..")

	ignore := map[string]bool{
		// Skip directories copied from other projects or vendored code.
		"internal/jsonschema": true,
		"internal/util":       true,
		"internal/httprr":     true,
		"vendor":              true,
	}

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if ignore[filepath.ToSlash(path)] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		hasHeader, err := hasCopyrightHeader(path)
		switch {
		case err != nil:
			t.Errorf("failed to check file %q: %v", path, err)
		case !hasHeader && !*fixError:
			t.Errorf("file %q does not have an accepted copyright header", path)
		case !hasHeader && *fixError:
			t.Logf("updating file %q with Archinfra copyright header", path)
			if err := addCopyrightHeader(path); err != nil {
				t.Errorf("failed to update file %q: %v", path, err)
			}
		}
		return nil
	})
}

func hasCopyrightHeader(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	contentStr := string(content)
	currentYear := time.Now().UTC().Year()

	for year := googleStartYear; year <= currentYear; year++ {
		if strings.HasPrefix(contentStr, fmt.Sprintf(googleCopyrightHeaderTmpl, year)) {
			return true, nil
		}
	}
	for year := archinfraStartYear; year <= currentYear; year++ {
		if strings.HasPrefix(contentStr, fmt.Sprintf(archinfraCopyrightHeaderTmpl, year)) {
			return true, nil
		}
	}
	return false, nil
}

func addCopyrightHeader(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	currentYearHeader := fmt.Sprintf(archinfraCopyrightHeaderTmpl, time.Now().UTC().Year())
	newContent := []byte(currentYearHeader)
	newContent = append(newContent, content...)
	return os.WriteFile(path, newContent, 0o644)
}
"@

$HeaderTestContent = $HeaderTestContent.Replace("__OWNER__", $Owner)
$HeaderTestContent = $HeaderTestContent.Replace("__GOOGLE_START_YEAR__", [string]$GoogleStartYear)
$HeaderTestContent = $HeaderTestContent.Replace("__CUSTOM_START_YEAR__", [string]$CustomStartYear)

Write-Utf8 $StyleTest $HeaderTestContent
Write-Host "Updated internal\style_test.go."

if (Test-Path $Golangci) {
  $Yml = Read-Utf8 $Golangci
  # Best-effort update for common goheader config values.
  $Yml = $Yml -replace 'COMPANY:\s*Google LLC', 'COMPANY: (Google LLC|Archinfra / yuanyp8)'
  $Yml = $Yml -replace 'copyright-holder:\s*Google LLC', 'copyright-holder: (Google LLC|Archinfra / yuanyp8)'
  $Yml = $Yml -replace 'Google LLC', 'Google LLC|Archinfra / yuanyp8'
  Write-Utf8 $Golangci $Yml
  Write-Host "Updated .golangci.yml if it had Google-only goheader values."
}

if (Test-Path $ModelLLM) {
  $ModelContent = Read-Utf8 $ModelLLM
  $OldDoc = "// Package model defines the interfaces and data structures for interacting with LLMs."
  $NewDoc = "// Package model defines the Archinfra Agent Runtime LLM abstraction layer. It contains the interfaces and data structures used to connect Gemini, OpenAI-compatible models, Qwen, DeepSeek, or other custom LLM providers."
  if ($ModelContent.Contains($OldDoc)) {
    $ModelContent = $ModelContent.Replace($OldDoc, $NewDoc)
    Write-Utf8 $ModelLLM $ModelContent
    Write-Host "Updated model\llm.go package documentation."
  } else {
    Write-Host "Skipped model\llm.go package documentation: original sentence not found."
  }
}

$NoticeContent = @"
# Archinfra / yuanyp8 Customization Notice

This repository is an Archinfra / yuanyp8 customized fork based on Google ADK Go.

- Custom fork owner: Archinfra / yuanyp8
- Original upstream copyright notices are preserved where they exist.
- New files generated for this fork should use:

```go
// Copyright 2026 Archinfra / yuanyp8
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
```

Recommended policy:

1. Preserve existing Google LLC headers on upstream-derived files.
2. Use Archinfra / yuanyp8 headers for new files or files that are primarily authored in this fork.
3. Keep the Apache-2.0 license text.
"@
Write-Utf8 $Notice $NoticeContent
Write-Host "Wrote NOTICE-ARCHINFRA.md."

if ($FixMissingHeaders) {
  $CustomHeader = @"
// Copyright 2026 Archinfra / yuanyp8
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

"@

  $Updated = 0
  Get-ChildItem -Path $Root -Recurse -File -Filter "*.go" | ForEach-Object {
    $file = $_.FullName
    if (Is-IgnoredPath $file) {
      return
    }

    $content = Read-Utf8 $file
    if (-not (Has-AcceptedHeader $content)) {
      Write-Utf8 $file ($CustomHeader + $content)
      $Updated++
      Write-Host ("Added Archinfra header: " + $_.FullName.Substring($Root.Length + 1))
    }
  }
  Write-Host ("Missing header fix completed. Updated files: " + $Updated)
}

Write-Host "Done. Recommended checks:"
Write-Host "  go test ./internal -run TestCopyrightHeader -count=1"
Write-Host "  go test ./... -run TestNotExist -count=0"
