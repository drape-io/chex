package config

// Config represents the complete chex configuration.
type Config struct {
	Chex  *ChexConfig
	Tools map[string]ToolConfig
}

// ChexConfig represents the [chex] section of the configuration.
type ChexConfig struct {
	Sources            []Source `toml:"sources"`
	FailOnUnknownTools bool     `toml:"fail_on_unknown_tools"` // Default: false
	SkipUnknownTools   bool     `toml:"skip_unknown_tools"`    // Default: false
	WarnOnUnknownTools bool     `toml:"warn_on_unknown_tools"` // Default: true
}

// Source represents an external configuration source.
type Source struct {
	Path string `toml:"path"`
	Type string `toml:"type"` // "chex", "mise", "tool-versions"
}

// ToolConfig represents a tool definition from the configuration file.
type ToolConfig struct {
	Name           string `toml:"name"`            // optional: override display name
	CLI            string `toml:"cli"`             // required: command to execute
	Version        string `toml:"version"`         // optional: version constraint
	VersionArg     string `toml:"version_arg"`     // optional: argument to get version
	VersionPattern string `toml:"version_pattern"` // optional: regex to extract version
	Optional       bool   `toml:"optional"`        // optional: mark as optional
	Message        string `toml:"message"`         // optional: custom message
}

// Tool represents a processed tool ready for checking.
type Tool struct {
	Name           string // display name
	CLI            string // command to execute
	Version        string // version constraint (empty = existence check only)
	VersionArg     string // argument to get version (default: "version" or "--version")
	VersionPattern string // regex to extract version
	Optional       bool   // whether tool is optional
	Message        string // custom message
	Source         string // where tool was defined ("config", "mise", "tool-versions")
}
