package repocontracts_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetupGoUsesGoModVersion(t *testing.T) {
	t.Parallel()

	for _, workflow := range []string{
		".github/workflows/ci.yml",
		".github/workflows/release.yml",
	} {
		workflow := workflow
		t.Run(workflow, func(t *testing.T) {
			t.Parallel()

			content := readRepoFile(t, workflow)
			setupGoCount := strings.Count(content, "uses: actions/setup-go@v6")
			if setupGoCount == 0 {
				t.Fatalf("%s does not configure actions/setup-go", workflow)
			}

			if got := strings.Count(content, "go-version-file: go.mod"); got != setupGoCount {
				t.Fatalf("%s should configure go-version-file for every setup-go step: got %d, want %d", workflow, got, setupGoCount)
			}

			if strings.Contains(content, "go-version: \"1.26.x\"") {
				t.Fatalf("%s still uses a wildcard go-version instead of go.mod", workflow)
			}
		})
	}
}

func readRepoFile(t *testing.T, relativePath string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	content, err := os.ReadFile(filepath.Join(repoRoot, relativePath))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}

	return string(content)
}
