package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("loads simple config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")

		content := `
[go]
cli = "go"
version = ">=1.20.0"

[docker]
cli = "docker"
version = ">=20.0.0"
optional = true
message = "Docker is optional"
`
		if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if len(cfg.Tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(cfg.Tools))
		}

		goTool := cfg.Tools["go"]
		if goTool.CLI != "go" {
			t.Errorf("expected CLI 'go', got %q", goTool.CLI)
		}
		if goTool.Version != ">=1.20.0" {
			t.Errorf("expected version '>=1.20.0', got %q", goTool.Version)
		}

		dockerTool := cfg.Tools["docker"]
		if !dockerTool.Optional {
			t.Error("expected docker to be optional")
		}
		if dockerTool.Message != "Docker is optional" {
			t.Errorf("expected message 'Docker is optional', got %q", dockerTool.Message)
		}
	})

	t.Run("loads config with chex section", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")

		content := `
[chex]
fail_on_unknown_tools = true
skip_unknown_tools = false

[[chex.sources]]
path = "mise.toml"
type = "mise"

[go]
cli = "go"
version = ">=1.20.0"
`
		if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Chex == nil {
			t.Fatal("expected Chex config to be loaded")
		}

		if !cfg.Chex.FailOnUnknownTools {
			t.Error("expected FailOnUnknownTools to be true")
		}

		if cfg.Chex.SkipUnknownTools {
			t.Error("expected SkipUnknownTools to be false")
		}

		if len(cfg.Chex.Sources) != 1 {
			t.Errorf("expected 1 source, got %d", len(cfg.Chex.Sources))
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := Load("/nonexistent/path/.chex.toml")
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("returns error for invalid TOML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")

		content := `
[go
invalid toml
`
		if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("expected error for invalid TOML")
		}
	})
}

func TestConfigToTool(t *testing.T) {
	t.Run("converts basic config", func(t *testing.T) {
		cfg := ToolConfig{
			CLI:     "go",
			Version: ">=1.20.0",
		}

		tool := configToTool("golang", cfg, "config")

		if tool.Name != "golang" {
			t.Errorf("expected name 'golang', got %q", tool.Name)
		}
		if tool.CLI != "go" {
			t.Errorf("expected CLI 'go', got %q", tool.CLI)
		}
		if tool.Version != ">=1.20.0" {
			t.Errorf("expected version '>=1.20.0', got %q", tool.Version)
		}
		if tool.Source != "config" {
			t.Errorf("expected source 'config', got %q", tool.Source)
		}
	})

	t.Run("uses custom name if provided", func(t *testing.T) {
		cfg := ToolConfig{
			Name: "Custom Name",
			CLI:  "go",
		}

		tool := configToTool("golang", cfg, "config")

		if tool.Name != "Custom Name" {
			t.Errorf("expected name 'Custom Name', got %q", tool.Name)
		}
	})
}

func TestExtractMiseVersion(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "simple string",
			value:    "1.20.0",
			expected: "1.20.0",
		},
		{
			name:     "map with version",
			value:    map[string]any{"version": "1.20.0"},
			expected: "1.20.0",
		},
		{
			name:     "map without version",
			value:    map[string]any{"other": "value"},
			expected: "",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMiseVersion(tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test helper functions
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func loadAndMergeHelper(t *testing.T, configPath, baseDir string) *LoadResult {
	t.Helper()
	result, err := LoadAndMerge(configPath, baseDir)
	if err != nil {
		t.Fatalf("LoadAndMerge() error = %v", err)
	}
	return result
}

func TestLoadAndMerge(t *testing.T) {
	t.Run("loads main config only", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")

		writeTestFile(t, configPath, `
[go]
cli = "go"
version = ">=1.20.0"
`)

		result := loadAndMergeHelper(t, configPath, tmpDir)

		if len(result.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(result.Tools))
		}

		if result.Tools["go"] == nil {
			t.Error("expected 'go' tool to be loaded")
		}
	})

	t.Run("auto-detects mise.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")
		misePath := filepath.Join(tmpDir, "mise.toml")

		writeTestFile(t, configPath, `# empty config`)
		writeTestFile(t, misePath, `
[tools]
golang = "1.20.0"
`)

		result := loadAndMergeHelper(t, configPath, tmpDir)

		if result.Tools["golang"] == nil {
			t.Error("expected 'golang' tool from mise.toml")
		}
	})

	t.Run("auto-detects .tool-versions", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")
		toolVersionsPath := filepath.Join(tmpDir, ".tool-versions")

		writeTestFile(t, configPath, `# empty config`)
		writeTestFile(t, toolVersionsPath, `golang 1.20.0
python 3.11.0
`)

		result := loadAndMergeHelper(t, configPath, tmpDir)

		if result.Tools["golang"] == nil {
			t.Error("expected 'golang' tool from .tool-versions")
		}
		if result.Tools["python"] == nil {
			t.Error("expected 'python' tool from .tool-versions")
		}
	})

	t.Run("main config overrides external sources", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")
		misePath := filepath.Join(tmpDir, "mise.toml")

		writeTestFile(t, configPath, `
[golang]
cli = "go"
version = ">=1.25.0"
`)
		writeTestFile(t, misePath, `
[tools]
golang = "1.20.0"
`)

		result := loadAndMergeHelper(t, configPath, tmpDir)

		goTool := result.Tools["golang"]
		if goTool == nil {
			t.Fatal("expected 'golang' tool")
		}

		if goTool.Version != ">=1.25.0" {
			t.Errorf("expected version '>=1.25.0', got %q", goTool.Version)
		}
	})

	t.Run("explicit chex source", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".chex.toml")
		externalPath := filepath.Join(tmpDir, "external.toml")

		writeTestFile(t, externalPath, `
[docker]
cli = "docker"
version = ">=20.0.0"
`)
		writeTestFile(t, configPath, `
[[chex.sources]]
path = "external.toml"
type = "chex"

[go]
cli = "go"
version = ">=1.20.0"
`)

		result := loadAndMergeHelper(t, configPath, tmpDir)

		if result.Tools["go"] == nil {
			t.Error("expected 'go' tool from main config")
		}
		if result.Tools["docker"] == nil {
			t.Error("expected 'docker' tool from external config")
		}
	})
}

func TestLoadMiseSource(t *testing.T) {
	t.Run("loads tools from mise.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		misePath := filepath.Join(tmpDir, "mise.toml")

		content := `
[tools]
golang = "1.20.0"
nodejs = { version = "18.0.0" }
`
		if err := os.WriteFile(misePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tools := make(map[string]*Tool)
		_ = loadMiseSource(misePath, tools, false, false, true)

		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}

		if tools["golang"] == nil {
			t.Error("expected golang tool")
		}
		if tools["golang"].Version != "1.20.0" {
			t.Errorf("expected version '1.20.0', got %q", tools["golang"].Version)
		}

		if tools["nodejs"] == nil {
			t.Error("expected nodejs tool")
		}
		if tools["nodejs"].Version != "18.0.0" {
			t.Errorf("expected version '18.0.0', got %q", tools["nodejs"].Version)
		}
	})

	t.Run("handles missing mise.toml", func(t *testing.T) {
		tools := make(map[string]*Tool)
		warnings := loadMiseSource("/nonexistent/mise.toml", tools, false, false, true)

		if len(warnings) != 0 {
			t.Errorf("expected no warnings for missing file, got %d", len(warnings))
		}
	})

	t.Run("respects failOnUnknown", func(t *testing.T) {
		tmpDir := t.TempDir()
		misePath := filepath.Join(tmpDir, "mise.toml")

		content := `
[tools]
unknown-tool-xyz = "1.0.0"
`
		if err := os.WriteFile(misePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tools := make(map[string]*Tool)
		warnings := loadMiseSource(misePath, tools, true, false, false)

		if len(warnings) == 0 {
			t.Error("expected warning for unknown tool with failOnUnknown")
		}
	})

	t.Run("respects skipUnknown", func(t *testing.T) {
		tmpDir := t.TempDir()
		misePath := filepath.Join(tmpDir, "mise.toml")

		content := `
[tools]
unknown-tool-xyz = "1.0.0"
golang = "1.20.0"
`
		if err := os.WriteFile(misePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tools := make(map[string]*Tool)
		_ = loadMiseSource(misePath, tools, false, true, false)

		if len(tools) != 1 {
			t.Errorf("expected 1 tool (unknown skipped), got %d", len(tools))
		}
		if tools["golang"] == nil {
			t.Error("expected golang tool to be loaded")
		}
	})
}

func TestLoadToolVersionsSource(t *testing.T) {
	t.Run("loads tools from .tool-versions", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolVersionsPath := filepath.Join(tmpDir, ".tool-versions")

		content := `golang 1.20.0
nodejs 18.0.0
# comment line
python 3.11.0

`
		if err := os.WriteFile(toolVersionsPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tools := make(map[string]*Tool)
		_ = loadToolVersionsSource(toolVersionsPath, tools, false, false, true)

		if len(tools) != 3 {
			t.Errorf("expected 3 tools, got %d", len(tools))
		}

		if tools["golang"] == nil {
			t.Error("expected golang tool")
		}
		if tools["golang"].Version != "1.20.0" {
			t.Errorf("expected version '1.20.0', got %q", tools["golang"].Version)
		}
	})

	t.Run("handles missing .tool-versions", func(t *testing.T) {
		tools := make(map[string]*Tool)
		warnings := loadToolVersionsSource("/nonexistent/.tool-versions", tools, false, false, true)

		if len(warnings) != 0 {
			t.Errorf("expected no warnings for missing file, got %d", len(warnings))
		}
	})

	t.Run("skips invalid lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		toolVersionsPath := filepath.Join(tmpDir, ".tool-versions")

		content := `golang 1.20.0
invalid-line-without-version

nodejs 18.0.0
`
		if err := os.WriteFile(toolVersionsPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tools := make(map[string]*Tool)
		loadToolVersionsSource(toolVersionsPath, tools, false, false, false)

		if len(tools) != 2 {
			t.Errorf("expected 2 tools (invalid line skipped), got %d", len(tools))
		}
	})
}

func TestLoadSource(t *testing.T) {
	t.Run("handles chex source type", func(t *testing.T) {
		tmpDir := t.TempDir()
		sourcePath := filepath.Join(tmpDir, "source.toml")

		content := `
[docker]
cli = "docker"
version = ">=20.0.0"
`
		if err := os.WriteFile(sourcePath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		tools := make(map[string]*Tool)
		warnings := loadSource(sourcePath, "chex", tools, false, false, false)

		if len(warnings) != 0 {
			t.Errorf("expected no warnings for chex source, got %d", len(warnings))
		}
		if len(tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("returns error for unknown source type", func(t *testing.T) {
		tools := make(map[string]*Tool)
		warnings := loadSource("/some/path", "unknown-type", tools, false, false, false)

		if len(warnings) == 0 {
			t.Error("expected warning for unknown source type")
		}
		if !strings.Contains(warnings[0], "unknown source type") {
			t.Errorf("expected 'unknown source type' in warning, got %q", warnings[0])
		}
	})
}
