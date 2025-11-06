package config

import (
	"testing"
)

func TestResolveToolMapping(t *testing.T) {
	tests := []struct {
		name            string
		toolName        string
		expectedCLI     string
		expectedVersion string
		expectedKnown   bool
	}{
		{
			name:            "nodejs maps to node",
			toolName:        "nodejs",
			expectedCLI:     "node",
			expectedVersion: "--version",
			expectedKnown:   true,
		},
		{
			name:            "golang maps to go",
			toolName:        "golang",
			expectedCLI:     "go",
			expectedVersion: "version",
			expectedKnown:   true,
		},
		{
			name:            "awscli maps to aws",
			toolName:        "awscli",
			expectedCLI:     "aws",
			expectedVersion: "--version",
			expectedKnown:   true,
		},
		{
			name:            "python is known",
			toolName:        "python",
			expectedCLI:     "python",
			expectedVersion: "--version",
			expectedKnown:   true,
		},
		{
			name:            "just is known",
			toolName:        "just",
			expectedCLI:     "just",
			expectedVersion: "--version",
			expectedKnown:   true,
		},
		{
			name:            "helm is known",
			toolName:        "helm",
			expectedCLI:     "helm",
			expectedVersion: "version",
			expectedKnown:   true,
		},
		{
			name:            "unknown tool",
			toolName:        "unknown-tool",
			expectedCLI:     "unknown-tool",
			expectedVersion: "",
			expectedKnown:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, versionArg, known := resolveToolMapping(tt.toolName)

			if cli != tt.expectedCLI {
				t.Errorf("expected CLI %q, got %q", tt.expectedCLI, cli)
			}

			if versionArg != tt.expectedVersion {
				t.Errorf("expected versionArg %q, got %q", tt.expectedVersion, versionArg)
			}

			if known != tt.expectedKnown {
				t.Errorf("expected known %v, got %v", tt.expectedKnown, known)
			}
		})
	}
}

func TestFormatUnknownToolWarning(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		source   string
		expected string
	}{
		{
			name:     "mise.toml warning",
			toolName: "sometool",
			source:   "mise.toml",
			expected: "Warning: Unknown tool 'sometool' from mise.toml. " +
				"Using 'sometool' as CLI command. " +
				"If this is incorrect, define it explicitly in .chex.toml",
		},
		{
			name:     ".tool-versions warning",
			toolName: "anothertool",
			source:   ".tool-versions",
			expected: "Warning: Unknown tool 'anothertool' from .tool-versions. " +
				"Using 'anothertool' as CLI command. " +
				"If this is incorrect, define it explicitly in .chex.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning := formatUnknownToolWarning(tt.toolName, tt.source)
			if warning != tt.expected {
				t.Errorf("expected warning:\n%s\ngot:\n%s", tt.expected, warning)
			}
		})
	}
}
