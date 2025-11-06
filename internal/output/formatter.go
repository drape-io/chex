package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/drape-io/chex/internal/checker"
	"github.com/fatih/color"
)

// Format represents the output format.
type Format string

const (
	FormatPretty Format = "pretty"
	FormatQuiet  Format = "quiet"
	FormatJSON   Format = "json"
)

// Print prints the check results in the specified format.
func Print(results []*checker.Result, format Format) {
	switch format {
	case FormatJSON:
		printJSON(results)
	case FormatQuiet:
		printQuiet(results)
	case FormatPretty:
		printPretty(results)
	default:
		printPretty(results)
	}
}

// printPretty prints results in a pretty colored format.
func printPretty(results []*checker.Result) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println("Checking CLI Tools...")
	fmt.Println()

	passed := 0
	failed := 0
	optionalMissing := 0

	for _, result := range results {
		tool := result.Tool

		// Print tool name with status
		switch result.Status {
		case checker.StatusPass:
			fmt.Printf("%s %s\n", green("✅"), tool.Name)
			passed++
		case checker.StatusFail:
			fmt.Printf("%s %s\n", red("❌"), tool.Name)
			failed++
		case checker.StatusOptionalMissing:
			fmt.Printf("%s %s (optional)\n", yellow("⚠️ "), tool.Name)
			optionalMissing++
		}

		// Print details
		if tool.Version != "" {
			// Version check
			if result.Output != "" {
				// Show command and output
				versionArg := tool.VersionArg
				if versionArg == "" {
					versionArg = "version"
				}
				fmt.Printf("   $ %s %s\n", tool.CLI, versionArg)
				firstLine := strings.Split(result.Output, "\n")[0]
				fmt.Printf("   %s\n", firstLine)
			}

			if result.Error != nil {
				fmt.Printf("   %s %s\n", red("Error:"), result.Error)
			}

			if tool.Version != "" {
				fmt.Printf("   Required: %s\n", tool.Version)
			}

			if result.InstalledVersion != "" {
				if result.Status == checker.StatusPass {
					fmt.Printf("   Installed: %s\n", green(result.InstalledVersion))
				} else {
					fmt.Printf("   Installed: %s\n", red(result.InstalledVersion))
				}
			}
		} else {
			// Existence check
			if result.Path != "" {
				fmt.Printf("   Found at: %s\n", cyan(result.Path))
			} else if result.Error != nil {
				fmt.Printf("   %s %s\n", red("Error:"), result.Error)
			}
		}

		// Print custom message if available
		if tool.Message != "" && result.Status != checker.StatusPass {
			fmt.Printf("   %s %s\n", cyan("Message:"), tool.Message)
		}

		fmt.Println()
	}

	// Print summary
	fmt.Printf(
		"Summary: %s passed, %s failed",
		green(strconv.Itoa(passed)),
		red(strconv.Itoa(failed)),
	)
	if optionalMissing > 0 {
		fmt.Printf(", %s optional missing", yellow(strconv.Itoa(optionalMissing)))
	}
	fmt.Println()
}

// printQuiet prints only failures in a compact format.
func printQuiet(results []*checker.Result) {
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	for _, result := range results {
		if result.Status == checker.StatusPass {
			continue
		}

		tool := result.Tool

		// Print tool name with status
		switch result.Status {
		case checker.StatusFail:
			fmt.Printf("%s %s\n", red("❌"), tool.Name)
		case checker.StatusOptionalMissing:
			fmt.Printf("%s %s (optional)\n", yellow("⚠️ "), tool.Name)
		case checker.StatusPass:
			// Already handled by continue above
		}

		// Print error
		if result.Error != nil {
			fmt.Printf("   %s %s\n", red("Error:"), result.Error)
		}

		// Print requirement
		if tool.Version != "" {
			fmt.Printf("   Required: %s\n", tool.Version)
		}

		fmt.Println()
	}
}

// printJSON prints results in JSON format.
func printJSON(results []*checker.Result) {
	passed := 0
	failed := 0
	optionalMissing := 0

	type JSONTool struct {
		Name             string `json:"name"`
		CLI              string `json:"cli"`
		Required         bool   `json:"required"`
		Status           string `json:"status"`
		VersionRequired  string `json:"versionRequired,omitempty"`
		VersionInstalled string `json:"versionInstalled,omitempty"`
		Command          string `json:"command,omitempty"`
		Output           string `json:"output,omitempty"`
		Path             string `json:"path,omitempty"`
		Error            string `json:"error,omitempty"`
		Message          string `json:"message,omitempty"`
	}

	type JSONOutput struct {
		Tools   []JSONTool `json:"tools"`
		Summary struct {
			Total           int `json:"total"`
			Passed          int `json:"passed"`
			Failed          int `json:"failed"`
			OptionalMissing int `json:"optionalMissing"`
		} `json:"summary"`
	}

	output := JSONOutput{}
	output.Tools = make([]JSONTool, 0, len(results))

	for _, result := range results {
		tool := result.Tool

		switch result.Status {
		case checker.StatusPass:
			passed++
		case checker.StatusFail:
			failed++
		case checker.StatusOptionalMissing:
			optionalMissing++
		}

		jsonTool := JSONTool{
			Name:             tool.Name,
			CLI:              tool.CLI,
			Required:         !tool.Optional,
			Status:           string(result.Status),
			VersionRequired:  tool.Version,
			VersionInstalled: result.InstalledVersion,
			Path:             result.Path,
			Message:          tool.Message,
		}

		if result.Error != nil {
			jsonTool.Error = result.Error.Error()
		}

		if result.Output != "" && tool.VersionArg != "" {
			jsonTool.Command = fmt.Sprintf("%s %s", tool.CLI, tool.VersionArg)
			jsonTool.Output = result.Output
		}

		output.Tools = append(output.Tools, jsonTool)
	}

	output.Summary.Total = len(results)
	output.Summary.Passed = passed
	output.Summary.Failed = failed
	output.Summary.OptionalMissing = optionalMissing

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// ShouldExitWithError determines if chex should exit with error code 1.
func ShouldExitWithError(results []*checker.Result) bool {
	for _, result := range results {
		if result.Status == checker.StatusFail {
			return true
		}
	}
	return false
}
