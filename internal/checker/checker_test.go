package checker

import (
	"strings"
	"testing"

	"github.com/drape-io/chex/internal/config"
)

func TestCheckExistence(t *testing.T) {
	t.Run("finds existing tool", func(t *testing.T) {
		tool := &config.Tool{
			Name: "go",
			CLI:  "go",
		}

		result := Check(tool)

		if result.Status != StatusPass {
			t.Errorf("expected StatusPass, got %v", result.Status)
		}
		if result.Path == "" {
			t.Error("expected path to be set")
		}
	})

	t.Run("fails for non-existent tool", func(t *testing.T) {
		tool := &config.Tool{
			Name: "nonexistent-tool-xyz",
			CLI:  "nonexistent-tool-xyz",
		}

		result := Check(tool)

		if result.Status != StatusFail {
			t.Errorf("expected StatusFail, got %v", result.Status)
		}
		if result.Error == nil {
			t.Error("expected error to be set")
		}
	})

	t.Run("optional missing tool", func(t *testing.T) {
		tool := &config.Tool{
			Name:     "nonexistent-tool-xyz",
			CLI:      "nonexistent-tool-xyz",
			Optional: true,
		}

		result := Check(tool)

		if result.Status != StatusOptionalMissing {
			t.Errorf("expected StatusOptionalMissing, got %v", result.Status)
		}
	})
}

func TestCheckVersion(t *testing.T) {
	t.Run("checks go version successfully", func(t *testing.T) {
		tool := &config.Tool{
			Name:    "go",
			CLI:     "go",
			Version: ">=1.0.0", // Very lenient to pass on any Go version
		}

		result := Check(tool)

		if result.Status != StatusPass {
			t.Errorf("expected StatusPass, got %v (error: %v)", result.Status, result.Error)
		}
		if result.InstalledVersion == "" {
			t.Error("expected installed version to be set")
		}
		if result.Output == "" {
			t.Error("expected output to be set")
		}
	})

	t.Run("fails for version mismatch", func(t *testing.T) {
		tool := &config.Tool{
			Name:    "go",
			CLI:     "go",
			Version: ">=999.0.0", // Impossible version
		}

		result := Check(tool)

		if result.Status != StatusFail {
			t.Errorf("expected StatusFail, got %v", result.Status)
		}
		if result.InstalledVersion == "" {
			t.Error("expected installed version to be set even on mismatch")
		}
	})

	t.Run("respects custom version arg", func(t *testing.T) {
		tool := &config.Tool{
			Name:       "go",
			CLI:        "go",
			Version:    ">=1.0.0",
			VersionArg: "version",
		}

		result := Check(tool)

		if result.Status != StatusPass {
			t.Errorf("expected StatusPass, got %v", result.Status)
		}
	})
}

func TestLooksLikeVersionOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "go version output",
			output:   "go version go1.20.0 darwin/amd64",
			expected: true,
		},
		{
			name:     "node version output",
			output:   "v18.16.0",
			expected: true,
		},
		{
			name:     "python version output",
			output:   "Python 3.11.0",
			expected: true,
		},
		{
			name:     "semver",
			output:   "1.2.3",
			expected: true,
		},
		{
			name:     "help output",
			output:   "Usage: tool [options]",
			expected: false,
		},
		{
			name:     "error output",
			output:   "Error: flag provided but not defined",
			expected: false,
		},
		{
			name:     "unknown flag",
			output:   "unknown flag: --version",
			expected: false,
		},
		{
			name:     "empty output",
			output:   "",
			expected: false,
		},
		{
			name:     "no digits",
			output:   "abcdef",
			expected: false,
		},
		{
			name:     "very long output",
			output:   strings.Repeat("a", 300),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeVersionOutput(tt.output)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		pattern        string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "go version",
			output:         "go version go1.20.0 darwin/amd64",
			pattern:        "",
			expectedResult: "1.20.0",
			expectError:    false,
		},
		{
			name:           "node version",
			output:         "v18.16.0",
			pattern:        "",
			expectedResult: "18.16.0",
			expectError:    false,
		},
		{
			name:           "python version",
			output:         "Python 3.11.4",
			pattern:        "",
			expectedResult: "3.11.4",
			expectError:    false,
		},
		{
			name:           "custom pattern",
			output:         "Version: v2.5.1",
			pattern:        `v(\d+\.\d+\.\d+)`,
			expectedResult: "2.5.1",
			expectError:    false,
		},
		{
			name:           "no version in output",
			output:         "No version information",
			pattern:        "",
			expectedResult: "",
			expectError:    true,
		},
		{
			name:           "pattern doesn't match",
			output:         "Version 1.0.0",
			pattern:        `v(\d+\.\d+)`,
			expectedResult: "",
			expectError:    true,
		},
		{
			name:           "multiline version output",
			output:         "Tool Name\nVersion 1.2.3\nMore info",
			pattern:        "",
			expectedResult: "1.2.3",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractVersion(tt.output, tt.pattern)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("expected %q, got %q", tt.expectedResult, result)
				}
			}
		})
	}
}

func TestCheckAll(t *testing.T) {
	t.Run("checks all tools", func(t *testing.T) {
		tools := map[string]*config.Tool{
			"go": {
				Name: "go",
				CLI:  "go",
			},
			"nonexistent": {
				Name: "nonexistent",
				CLI:  "nonexistent-tool-xyz",
			},
		}

		results := CheckAll(tools, nil)

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		// Check that we have one pass and one fail
		passCount := 0
		failCount := 0
		for _, r := range results {
			switch r.Status {
			case StatusPass:
				passCount++
			case StatusFail:
				failCount++
			case StatusOptionalMissing:
				// Not counted in this test
			}
		}

		if passCount != 1 {
			t.Errorf("expected 1 pass, got %d", passCount)
		}
		if failCount != 1 {
			t.Errorf("expected 1 fail, got %d", failCount)
		}
	})

	t.Run("filters tools", func(t *testing.T) {
		tools := map[string]*config.Tool{
			"go": {
				Name: "go",
				CLI:  "go",
			},
			"python": {
				Name: "python",
				CLI:  "python",
			},
		}

		results := CheckAll(tools, []string{"go"})

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		if results[0].Tool.Name != "go" {
			t.Errorf("expected go result, got %s", results[0].Tool.Name)
		}
	})

	t.Run("returns error for unknown tool", func(t *testing.T) {
		tools := map[string]*config.Tool{
			"go": {
				Name: "go",
				CLI:  "go",
			},
		}

		results := CheckAll(tools, []string{"unknown"})

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		if results[0].Status != StatusFail {
			t.Errorf("expected StatusFail, got %v", results[0].Status)
		}

		if results[0].Error == nil {
			t.Error("expected error for unknown tool")
		}
	})
}
