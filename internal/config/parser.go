package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Load loads and parses the chex configuration from the specified path.
// If path is empty, it searches for .chex.toml in the current directory.
func Load(path string) (*Config, error) {
	if path == "" {
		path = ".chex.toml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// First decode into a generic map to get all sections
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg := &Config{
		Tools: make(map[string]ToolConfig),
	}

	// Process each section
	for name, value := range raw {
		if name == "chex" {
			// Parse chex config section
			var chexCfg ChexConfig
			err := decodeInto(value, &chexCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to parse [chex] section: %w", err)
			}
			cfg.Chex = &chexCfg
		} else {
			// Parse as tool config
			var toolCfg ToolConfig
			err := decodeInto(value, &toolCfg)
			if err != nil {
				return nil, fmt.Errorf("failed to parse [%s] section: %w", name, err)
			}
			cfg.Tools[name] = toolCfg
		}
	}

	return cfg, nil
}

// decodeInto decodes a generic interface{} into a target struct.
func decodeInto(from any, to any) error {
	// Convert to TOML and back to decode properly
	data, err := toml.Marshal(from)
	if err != nil {
		return err
	}
	return toml.Unmarshal(data, to)
}

// LoadResult contains the loaded tools and any warnings.
type LoadResult struct {
	Tools    map[string]*Tool
	Warnings []string
}

// LoadAndMerge loads the main config and merges external sources.
func LoadAndMerge(path, rootDir string) (*LoadResult, error) {
	if path == "" {
		path = ".chex.toml"
	}
	if rootDir == "" {
		rootDir = "."
	}

	// Load main config - only use rootDir if path is relative
	configPath := path
	if !filepath.IsAbs(path) {
		configPath = filepath.Join(rootDir, path)
	}
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}

	result := &LoadResult{
		Tools:    make(map[string]*Tool),
		Warnings: []string{},
	}

	// Convert config tools to Tool structs
	for name, toolCfg := range cfg.Tools {
		tool := configToTool(name, toolCfg, "config")
		result.Tools[name] = &tool
	}

	// Determine behavior for unknown tools
	failOnUnknown := cfg.Chex != nil && cfg.Chex.FailOnUnknownTools
	skipUnknown := cfg.Chex != nil && cfg.Chex.SkipUnknownTools
	warnOnUnknown := cfg.Chex == nil || cfg.Chex.WarnOnUnknownTools // Default: true

	// Load external sources
	if cfg.Chex != nil && len(cfg.Chex.Sources) > 0 {
		// Explicit sources defined
		for _, source := range cfg.Chex.Sources {
			// Only use rootDir if source path is relative
			sourcePath := source.Path
			if !filepath.IsAbs(source.Path) {
				sourcePath = filepath.Join(rootDir, source.Path)
			}
			warnings := loadSource(
				sourcePath,
				source.Type,
				result.Tools,
				failOnUnknown,
				skipUnknown,
				warnOnUnknown,
			)
			result.Warnings = append(result.Warnings, warnings...)
		}
	} else {
		// Auto-detect mise.toml and .tool-versions (always relative to rootDir)
		misePath := filepath.Join(rootDir, "mise.toml")
		if _, err := os.Stat(misePath); err == nil {
			warnings := loadSource(misePath, "mise", result.Tools, failOnUnknown, skipUnknown, warnOnUnknown)
			result.Warnings = append(result.Warnings, warnings...)
		}

		toolVersionsPath := filepath.Join(rootDir, ".tool-versions")
		if _, err := os.Stat(toolVersionsPath); err == nil {
			warnings := loadSource(toolVersionsPath, "tool-versions", result.Tools, failOnUnknown, skipUnknown, warnOnUnknown)
			result.Warnings = append(result.Warnings, warnings...)
		}
	}

	return result, nil
}

// configToTool converts a ToolConfig to a Tool.
func configToTool(name string, cfg ToolConfig, source string) Tool {
	displayName := name
	if cfg.Name != "" {
		displayName = cfg.Name
	}

	// Don't set a default version arg - let smart guessing handle it
	versionArg := cfg.VersionArg

	return Tool{
		Name:           displayName,
		CLI:            cfg.CLI,
		Version:        cfg.Version,
		VersionArg:     versionArg,
		VersionPattern: cfg.VersionPattern,
		Optional:       cfg.Optional,
		Message:        cfg.Message,
		Source:         source,
	}
}

// loadSource loads tools from an external source and merges them into the tools map.
// It doesn't override tools that are already defined in the main config.
// Returns warnings about unknown tools.
func loadSource(
	path, sourceType string,
	tools map[string]*Tool,
	failOnUnknown, skipUnknown, warnOnUnknown bool,
) []string {
	switch sourceType {
	case "chex":
		// chex sources don't have unknown tools
		_ = loadChexSource(path, tools)
		return nil
	case "mise":
		return loadMiseSource(path, tools, failOnUnknown, skipUnknown, warnOnUnknown)
	case "tool-versions":
		return loadToolVersionsSource(path, tools, failOnUnknown, skipUnknown, warnOnUnknown)
	default:
		return []string{"Error: unknown source type: " + sourceType}
	}
}

// loadChexSource loads tools from another chex config file.
func loadChexSource(path string, tools map[string]*Tool) error {
	cfg, err := Load(path)
	if err != nil {
		return err
	}

	for name, toolCfg := range cfg.Tools {
		// Don't override existing tools
		if _, exists := tools[name]; exists {
			continue
		}
		tool := configToTool(name, toolCfg, "chex:"+path)
		tools[name] = &tool
	}

	return nil
}

// loadMiseSource loads tools from a mise.toml file.
func loadMiseSource(
	path string,
	tools map[string]*Tool,
	failOnUnknown, skipUnknown, warnOnUnknown bool,
) []string {
	var warnings []string

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist, skip silently
		return nil
	}

	type MiseConfig struct {
		Tools map[string]any `toml:"tools"`
	}

	var miseCfg MiseConfig
	if err := toml.Unmarshal(data, &miseCfg); err != nil {
		return []string{fmt.Sprintf("Error parsing mise.toml: %v", err)}
	}

	for name, value := range miseCfg.Tools {
		// Don't override existing tools from main config
		if _, exists := tools[name]; exists {
			continue
		}

		// Extract version from various mise.toml formats
		version := extractMiseVersion(value)

		// Resolve tool mapping
		cli, versionArg, known := resolveToolMapping(name)

		// Handle unknown tools
		if !known {
			if skipUnknown {
				continue
			}
			if failOnUnknown {
				warnings = append(
					warnings,
					fmt.Sprintf("Error: Unknown tool '%s' in mise.toml", name),
				)
				continue
			}
			if warnOnUnknown {
				warnings = append(warnings, formatUnknownToolWarning(name, "mise.toml"))
			}
		}

		tool := &Tool{
			Name:       name,
			CLI:        cli,
			Version:    version,
			VersionArg: versionArg,
			Optional:   false,
			Source:     "mise:" + path,
		}

		tools[name] = tool
	}

	return warnings
}

// extractMiseVersion extracts version from various mise.toml value types.
func extractMiseVersion(value any) string {
	switch v := value.(type) {
	case string:
		// Simple string version: node = "18"
		return v
	case map[string]any:
		// Object with version: terraform = {version="1.0.0"}
		if ver, ok := v["version"].(string); ok {
			return ver
		}
	}
	return ""
}

// loadToolVersionsSource loads tools from a .tool-versions file.
func loadToolVersionsSource(
	path string,
	tools map[string]*Tool,
	failOnUnknown, skipUnknown, warnOnUnknown bool,
) []string {
	var warnings []string

	file, err := os.Open(path)
	if err != nil {
		// File doesn't exist, skip silently
		return nil
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse space-separated format: tool_name version
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		version := parts[1]

		// Don't override existing tools from main config
		if _, exists := tools[name]; exists {
			continue
		}

		// Resolve tool mapping
		cli, versionArg, known := resolveToolMapping(name)

		// Handle unknown tools
		if !known {
			if skipUnknown {
				continue
			}
			if failOnUnknown {
				warnings = append(
					warnings,
					fmt.Sprintf("Error: Unknown tool '%s' in .tool-versions", name),
				)
				continue
			}
			if warnOnUnknown {
				warnings = append(warnings, formatUnknownToolWarning(name, ".tool-versions"))
			}
		}

		tool := &Tool{
			Name:       name,
			CLI:        cli,
			Version:    version,
			VersionArg: versionArg,
			Optional:   false,
			Source:     "tool-versions:" + path,
		}

		tools[name] = tool
	}

	if err := scanner.Err(); err != nil {
		warnings = append(warnings, fmt.Sprintf("Error reading .tool-versions: %v", err))
	}

	return warnings
}
