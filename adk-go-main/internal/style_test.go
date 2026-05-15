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

const licenseText = `//
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
`

const (
	validStartYear = 2025
	archinfraOwner = "Archinfra / yuanyp8"
	googleOwner    = "Google LLC"
)

var fixError = flag.Bool("fix", false, "fix detected problems, such as adding missing copyright headers")

func TestCopyrightHeader(t *testing.T) {
	// Start test from the parent directory, root of the module.
	t.Chdir("..")

	ignore := map[string]bool{
		// Copied or vendored code.
		"internal/jsonschema": true,
		"internal/util":       true,
		"internal/httprr":     true,
		"vendor":              true,
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			cleanPath := filepath.ToSlash(path)
			if ignore[cleanPath] {
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
	if err != nil {
		t.Fatalf("failed to walk repository: %v", err)
	}
}

func hasCopyrightHeader(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	contentStr := string(content)
	currentYear := time.Now().UTC().Year()

	for year := validStartYear; year <= currentYear; year++ {
		acceptedHeaders := []string{
			copyrightHeader(year, googleOwner),
			copyrightHeader(year, archinfraOwner),
		}

		for _, expectedHeader := range acceptedHeaders {
			if strings.HasPrefix(contentStr, expectedHeader) {
				return true, nil
			}
		}
	}

	return false, nil
}

func addCopyrightHeader(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	currentYearHeader := copyrightHeader(time.Now().UTC().Year(), archinfraOwner)
	newContent := []byte(currentYearHeader)
	newContent = append(newContent, content...)

	return os.WriteFile(path, newContent, 0o644)
}

func copyrightHeader(year int, owner string) string {
	return fmt.Sprintf("// Copyright %d %s\n%s", year, owner, licenseText)
}
