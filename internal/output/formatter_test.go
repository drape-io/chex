package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/drape-io/chex/internal/checker"
	"github.com/drape-io/chex/internal/config"
)

func TestPrint(t *testing.T) {
	results := []*checker.Result{
		{
			Tool: &config.Tool{
				Name:    "go",
				CLI:     "go",
				Version: ">=1.20.0",
			},
			Status:           checker.StatusPass,
			InstalledVersion: "1.25.4",
			Output:           "go version go1.25.4 darwin/amd64",
		},
	}

	t.Run("pretty format", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Print(results, FormatPretty)

		_ = w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "go") {
			t.Error("expected output to contain 'go'")
		}
		if !strings.Contains(output, "Summary") {
			t.Error("expected output to contain 'Summary'")
		}
	})

	t.Run("json format", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Print(results, FormatJSON)

		_ = w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		// Verify it's valid JSON
		var jsonOutput map[string]any
		if err := json.Unmarshal([]byte(output), &jsonOutput); err != nil {
			t.Errorf("expected valid JSON, got error: %v", err)
		}

		// Check structure
		if jsonOutput["tools"] == nil {
			t.Error("expected 'tools' key in JSON output")
		}
		if jsonOutput["summary"] == nil {
			t.Error("expected 'summary' key in JSON output")
		}
	})

	t.Run("quiet format", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Print(results, FormatQuiet)

		_ = w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		// Quiet format should show nothing for passing tools
		if strings.Contains(output, "go") {
			t.Error("expected quiet format to not show passing tools")
		}
	})
}

func TestPrintQuiet(t *testing.T) {
	t.Run("shows only failures", func(t *testing.T) {
		results := []*checker.Result{
			{
				Tool: &config.Tool{
					Name: "go",
					CLI:  "go",
				},
				Status: checker.StatusPass,
			},
			{
				Tool: &config.Tool{
					Name:    "docker",
					CLI:     "docker",
					Version: ">=20.0.0",
				},
				Status: checker.StatusFail,
				Error:  errors.New("version mismatch"),
			},
		}

		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		printQuiet(results)

		_ = w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if strings.Contains(output, "go") {
			t.Error("expected quiet output to not show passing tools")
		}
		if !strings.Contains(output, "docker") {
			t.Error("expected quiet output to show failing tools")
		}
	})
}

func TestPrintJSON(t *testing.T) {
	t.Run("outputs valid JSON", func(t *testing.T) {
		results := []*checker.Result{
			{
				Tool: &config.Tool{
					Name:    "go",
					CLI:     "go",
					Version: ">=1.20.0",
				},
				Status:           checker.StatusPass,
				InstalledVersion: "1.25.4",
			},
			{
				Tool: &config.Tool{
					Name:     "docker",
					CLI:      "docker",
					Version:  ">=20.0.0",
					Optional: true,
				},
				Status: checker.StatusOptionalMissing,
				Error:  errors.New("not found"),
			},
		}

		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		printJSON(results)

		_ = w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		// Parse JSON
		var jsonOutput struct {
			Tools []struct {
				Name             string `json:"name"`
				CLI              string `json:"cli"`
				Required         bool   `json:"required"`
				Status           string `json:"status"`
				VersionRequired  string `json:"versionRequired,omitempty"`
				VersionInstalled string `json:"versionInstalled,omitempty"`
				Error            string `json:"error,omitempty"`
			} `json:"tools"`
			Summary struct {
				Total           int `json:"total"`
				Passed          int `json:"passed"`
				Failed          int `json:"failed"`
				OptionalMissing int `json:"optionalMissing"`
			} `json:"summary"`
		}

		if err := json.Unmarshal([]byte(output), &jsonOutput); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Verify structure
		if len(jsonOutput.Tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(jsonOutput.Tools))
		}

		if jsonOutput.Summary.Total != 2 {
			t.Errorf("expected total 2, got %d", jsonOutput.Summary.Total)
		}

		if jsonOutput.Summary.Passed != 1 {
			t.Errorf("expected 1 passed, got %d", jsonOutput.Summary.Passed)
		}

		if jsonOutput.Summary.OptionalMissing != 1 {
			t.Errorf("expected 1 optional missing, got %d", jsonOutput.Summary.OptionalMissing)
		}

		// Verify camelCase field names
		if !strings.Contains(output, "versionRequired") {
			t.Error("expected camelCase 'versionRequired' in JSON")
		}
		if !strings.Contains(output, "versionInstalled") {
			t.Error("expected camelCase 'versionInstalled' in JSON")
		}
		if !strings.Contains(output, "optionalMissing") {
			t.Error("expected camelCase 'optionalMissing' in JSON")
		}
	})
}

func TestShouldExitWithError(t *testing.T) {
	tests := []struct {
		name     string
		results  []*checker.Result
		expected bool
	}{
		{
			name: "all pass",
			results: []*checker.Result{
				{
					Status: checker.StatusPass,
				},
			},
			expected: false,
		},
		{
			name: "one failure",
			results: []*checker.Result{
				{
					Status: checker.StatusPass,
				},
				{
					Status: checker.StatusFail,
				},
			},
			expected: true,
		},
		{
			name: "optional missing but no failures",
			results: []*checker.Result{
				{
					Status: checker.StatusPass,
				},
				{
					Status: checker.StatusOptionalMissing,
				},
			},
			expected: false,
		},
		{
			name:     "empty results",
			results:  []*checker.Result{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldExitWithError(tt.results)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
