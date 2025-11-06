package config

import "fmt"

// ToolMapping defines how to interact with a tool from mise/asdf.
type ToolMapping struct {
	CLI        string // The actual CLI command
	VersionArg string // Argument to get version (empty means auto-detect)
}

// knownToolMappings maps mise/asdf tool names to their CLI details.
// Includes version args to avoid guessing and prevent unknown tool warnings.
var knownToolMappings = map[string]ToolMapping{
	"nodejs":        {CLI: "node", VersionArg: "--version"},
	"golang":        {CLI: "go", VersionArg: "version"},
	"awscli":        {CLI: "aws", VersionArg: "--version"},
	"golangci-lint": {CLI: "golangci-lint", VersionArg: "version"},
	"just":          {CLI: "just", VersionArg: "--version"},
	"pnpm":          {CLI: "pnpm", VersionArg: "--version"},
	"tilt":          {CLI: "tilt", VersionArg: "version"},
	"python":        {CLI: "python", VersionArg: "--version"},
	"poetry":        {CLI: "poetry", VersionArg: "--version"},
	"helm":          {CLI: "helm", VersionArg: "version"},
	"kustomize":     {CLI: "kustomize", VersionArg: "version"},
	"mockery":       {CLI: "mockery", VersionArg: "version"},
	"kubeconform":   {CLI: "kubeconform", VersionArg: "version"},
}

// resolveToolMapping resolves a tool name from mise/asdf to CLI command and version arg.
// Returns the mapping and a boolean indicating if it's a known tool.
func resolveToolMapping(name string) (cli string, versionArg string, known bool) {
	if mapping, exists := knownToolMappings[name]; exists {
		return mapping.CLI, mapping.VersionArg, true
	}

	// Unknown tool: use name as-is for CLI, empty version arg (will try defaults)
	return name, "", false
}

// formatUnknownToolWarning formats a warning message for unknown tools.
func formatUnknownToolWarning(name, source string) string {
	return fmt.Sprintf(
		"Warning: Unknown tool '%s' from %s. Using '%s' as CLI command. "+
			"If this is incorrect, define it explicitly in .chex.toml",
		name, source, name,
	)
}
