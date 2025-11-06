package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/drape-io/chex/internal/checker"
	"github.com/drape-io/chex/internal/config"
	"github.com/drape-io/chex/internal/output"
	"github.com/spf13/cobra"
)

var (
	configFile   string
	quiet        bool
	outputFormat string
	rootDir      string
	version      = "dev" // Will be set by build
)

var rootCmd = &cobra.Command{
	Use:   "chex [tool-names...]",
	Short: "Check CLI tool versions",
	Long: `chex verifies that required CLI tools are installed and match version requirements.

Examples:
  chex                    # Check all tools
  chex go docker          # Check only go and docker
  chex --output=json      # Output in JSON format`,
	RunE:               runCheck,
	DisableFlagParsing: false,
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	Args:               cobra.ArbitraryArgs, // Allow any positional arguments
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: false},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a sample .chex.toml configuration file",
	RunE:  runInit,
}

func init() {
	rootCmd.Flags().StringVar(&configFile, "config", "", "config file (default: .chex.toml)")
	rootCmd.Flags().BoolVar(&quiet, "quiet", false, "only show failures")
	rootCmd.Flags().StringVar(
		&outputFormat,
		"output",
		"pretty",
		"output format (pretty|quiet|json)",
	)
	rootCmd.Flags().StringVar(&rootDir, "root", ".", "root directory to search for config")

	rootCmd.AddCommand(initCmd)
	rootCmd.Version = version
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Load and merge configurations
	loadResult, err := config.LoadAndMerge(configFile, rootDir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Display warnings
	for _, warning := range loadResult.Warnings {
		fmt.Fprintln(os.Stderr, warning)
	}
	if len(loadResult.Warnings) > 0 {
		fmt.Fprintln(os.Stderr)
	}

	if len(loadResult.Tools) == 0 {
		return errors.New("no tools defined in configuration")
	}

	// Check tools (with optional filter)
	results := checker.CheckAll(loadResult.Tools, args)

	// Check if any specified tool was not found
	if len(args) > 0 {
		for _, result := range results {
			if result.Error != nil &&
				result.Error.Error() == fmt.Sprintf(
					"tool '%s' not found in configuration",
					result.Tool.Name,
				) {
				// List available tools
				fmt.Fprintf(
					os.Stderr,
					"Error: tool '%s' not found in configuration\n\n",
					result.Tool.Name,
				)
				fmt.Fprintln(os.Stderr, "Available tools:")
				for name := range loadResult.Tools {
					fmt.Fprintf(os.Stderr, "  - %s\n", name)
				}
				return errors.New("invalid tool name")
			}
		}
	}

	// Determine output format
	outFormat := output.FormatPretty
	if quiet {
		outFormat = output.FormatQuiet
	} else {
		switch outputFormat {
		case "json":
			outFormat = output.FormatJSON
		case "quiet":
			outFormat = output.FormatQuiet
		case "pretty":
			outFormat = output.FormatPretty
		}
	}

	// Print results
	output.Print(results, outFormat)

	// Exit with error code if any checks failed
	if output.ShouldExitWithError(results) {
		os.Exit(1)
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if .chex.toml already exists
	if _, err := os.Stat(".chex.toml"); err == nil {
		return errors.New(".chex.toml already exists")
	}

	// Generate sample configuration
	sample := `# chex configuration file
# Check CLI tool versions

# Optional: configure external sources
# [chex]
# sources = [
#   { path = "mise.toml", type = "mise" },
#   { path = ".tool-versions", type = "tool-versions" }
# ]

[go]
cli = "go"
version = ">=1.20.0"

[docker]
cli = "docker"
version = ">=20.0.0"
optional = true
message = "Docker is optional but recommended for containerized development"

[node]
name = "Node.js"
cli = "node"
version = "^18.0.0 || ^20.0.0"
version_arg = "-v"

[make]
cli = "make"
# No version specified = existence check only
`

	err := os.WriteFile(".chex.toml", []byte(sample), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write .chex.toml: %w", err)
	}

	fmt.Println("Created .chex.toml")
	return nil
}
